package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/internal/httputil"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/Kostaaa1/twitch/pkg/twitch/usher"
)

type Transfer struct {
	// max number of segments that can be fetched ahead accross all units
	// if specified it will be used instead of MaxReadAheadPerUnit
	MaxReadAheadGlobal int
	// max number of segments that can be fetched ahead per unit (prevent unlimited ahead fetching)
	MaxReadAheadPerUnit int
	// each unit spawns N amount of workers that fetches the segments
	MaxWorkersPerUnit int
	// if twitch rate limits us
	MaxRetries       int
	ExpBackoffPeriod int
}

func defaultTransfer() *Transfer {
	return &Transfer{
		MaxReadAheadGlobal:  0,
		MaxReadAheadPerUnit: 32,
		MaxWorkersPerUnit:   4,
	}
}

type Downloader struct {
	gql          *gql.Client
	http         *http.Client
	usher        *usher.Client
	notifyFn     func(Progress)
	transfer     *Transfer
	retryMuteSeg bool
}

func New(gql *gql.Client, http *http.Client) *Downloader {
	return &Downloader{
		usher:        usher.New(http, gql),
		gql:          gql,
		http:         http,
		retryMuteSeg: true,
		transfer:     defaultTransfer(),
	}
}

func (dl *Downloader) SetProgressNotifier(fn func(Progress)) {
	dl.notifyFn = fn
}

func (dl *Downloader) Download(ctx context.Context, u *Unit) error {
	defer u.CloseWriter()

	if u.Error != nil {
		dl.notifyDone(u)
		return u.Error
	}

	var err error

	switch u.Type {
	case TypeVOD:
		err = dl.downloadVideo(ctx, u)
	case TypeClip:
		err = dl.downloadClip(ctx, u)
	case TypeLivestream:
		err = dl.recordLivestream(ctx, u)
	}

	u.Error = err
	dl.notifyDone(u)

	return u.Error
}

func (dl *Downloader) fetchTitle(ctx context.Context, u *Unit) (title string, err error) {
	switch u.Type {
	case TypeClip:
		title, err = dl.gql.ClipTitle(ctx, u.ID)
	case TypeVOD:
		title, err = dl.gql.VideoTitle(ctx, u.ID)
	case TypeLivestream:
		title, err = dl.gql.StreamTitle(ctx, u.ID)
	}
	return
}

func (dl *Downloader) openFile(ctx context.Context, u *Unit) error {
	if u.dir == "" {
		return errors.New("missing dir")
	}
	if u.ext == "" {
		return errors.New("missing file extension")
	}

	if u.filename == "" {
		title, err := dl.fetchTitle(ctx, u)
		if err != nil {
			return err
		}
		u.filename = title
	}

	pathname, err := fileutil.ConstructPathname(u.dir, u.filename, u.ext)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(pathname, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	u.w = f

	return nil
}

func (dl *Downloader) downloadBytes(ctx context.Context, u *Unit, b []byte) error {
	if u.w == nil {
		if err := dl.openFile(ctx, u); err != nil {
			return err
		}
	}
	n, err := u.w.Write(b)
	if err != nil {
		return err
	}
	dl.notifyProgress(u, int64(n))
	return nil
}

func (dl *Downloader) download(ctx context.Context, u *Unit, r io.ReadCloser) error {
	defer r.Close()

	if u.w == nil {
		if err := dl.openFile(ctx, u); err != nil {
			return err
		}
	}

	n, err := io.Copy(u.w, r)
	if err != nil {
		return err
	}

	dl.notifyProgress(u, n)

	return nil
}

func (dl *Downloader) fetchDownload(ctx context.Context, u *Unit, segURL string) error {
	body, err := dl.fetchSegment(ctx, segURL)
	if err != nil {
		return err
	}
	return dl.download(ctx, u, body)
}

// this is called when 403 occurs (meaning the url failed to download). used when retrying to recover the unmuted segments (if unit VOD audio is recoverable). n-muted.ts should be the output for the last try
// init-0.ts -> init-0.ts
// n.ts -> n-unmuted.ts
// n-unmuted.ts -> n-muted.ts
// n-muted.ts -> n-muted.ts
func transformSegmentURL(url string) (string, bool) {
	if strings.LastIndex(filepath.Base(url), "-") == -1 {
		ext := filepath.Ext(url)
		return strings.TrimSuffix(url, ext) + "-unmuted" + ext, false
	}

	replaced := strings.Replace(url, "-unmuted", "-muted", 1)
	if replaced != url {
		return replaced, false
	}

	return url, true
}

func stripSegmentURLType(url string) string {
	url = strings.Replace(url, "-unmuted", "", 1)
	url = strings.Replace(url, "-muted", "", 1)
	return url
}

// segment URLs can be structured like this: 0.ts, 0-muted.ts, 0-unmuted.ts. Twitch will mute certain segments because of DMCA (0-muted.ts). Audio from these segments can be recovered if they are fetched within a short period from the original livestream. So we automatically try to fetch unmuted segments.
// Also, we do not want to do this for all (older) videos
func (dl *Downloader) fetchSegment(ctx context.Context, url string) (io.ReadCloser, error) {
	url = stripSegmentURLType(url)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			resp, err := httputil.Do(ctx, dl.http, url, http.MethodGet, nil, nil)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode == http.StatusForbidden {
				u, done := transformSegmentURL(url)
				if done {
					panic(fmt.Errorf("got 403 error for -muted segment: %s", url))
				}
				url = u
				continue
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				b, err := io.ReadAll(resp.Body)
				resp.Body.Close()

				if err != nil {
					return nil, fmt.Errorf("failed to read the error response: %v", err)
				}

				return nil, fmt.Errorf(
					"failed to fetch segment - invalid status code %d: response: %s", resp.StatusCode, string(b),
				)
			}

			return resp.Body, nil
		}
	}
}
