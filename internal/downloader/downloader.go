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
)

type Transfer struct {
	// max number of segments fetched accross all units
	MaxReadAheadGlobal int
	// max number of segments fetched per unit - discarded if MaxReadSegmentsGlobal > 0 (prevents unlimited fetching ahead)
	MaxReadAheadPerUnit int
	// each unit spawns N amount of workers that fetches the segments
	MaxWorkersPerUnit int
	// flag that disables segment stripping (-muted, -unmuted), meaning program will not try to recover muted segments by trying to fetch unmuted first
	DisableSegmentRetries bool
}

func defaultTransfer() *Transfer {
	return &Transfer{
		MaxReadAheadGlobal:    0,
		MaxReadAheadPerUnit:   32,
		MaxWorkersPerUnit:     4,
		DisableSegmentRetries: false,
	}
}

type Downloader struct {
	gql      *gql.Client
	http     *http.Client
	notifyFn func(Progress)
	transfer *Transfer
}

func New(gql *gql.Client, http *http.Client) *Downloader {
	return &Downloader{
		gql:      gql,
		http:     http,
		transfer: defaultTransfer(),
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
		err = dl.downloadVOD(ctx, u)
	case TypeClip:
		err = dl.downloadClip(ctx, u)
	case TypeLivestream:
		err = dl.recordLivestream(ctx, u)
	}

	u.Error = err
	dl.notifyDone(u)

	return u.Error
}

func (dl *Downloader) fetchDownload(ctx context.Context, u *Unit, segURL string) error {
	body, err := dl.fetchSegment(ctx, u, segURL)
	if err != nil {
		return err
	}
	return dl.download(u, body)
}

func (dl *Downloader) downloadBytes(u *Unit, b []byte) error {
	if u.w == nil {
		if err := dl.openFile(context.Background(), u); err != nil {
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

func (dl *Downloader) download(u *Unit, r io.ReadCloser) error {
	defer r.Close()
	if u.w == nil {
		if err := dl.openFile(context.Background(), u); err != nil {
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
func (dl *Downloader) fetchSegment(ctx context.Context, u *Unit, url string) (io.ReadCloser, error) {
	if u.recoverAudio.Load() {
		url = stripSegmentURLType(url)
	}

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

			// if success with muted in url, means that segment is not recoverable
			if u.recoverAudio.Load() && strings.Contains(url, "-muted") {
				u.recoverAudio.Store(false)
			}

			return resp.Body, nil
		}
	}
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

// TODO: should not depend on fileutil
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
		title = fmt.Sprintf("%s_%s", title, u.Quality.String())
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
