package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
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
		MaxReadAheadPerUnit: 32,
		MaxWorkersPerUnit:   4,
		MaxReadAheadGlobal:  0,
	}
}

type Downloader struct {
	gql          *gql.Client
	http         *http.Client
	usher        *usher.Client
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

func (dl *Downloader) Download(ctx context.Context, u *Unit) error {
	defer u.CloseWriter()

	if u.Error != nil {
		u.notifyDone()
		return u.Error
	}

	switch u.Type {
	case TypeVOD:
		u.Error = dl.downloadVideo(ctx, u)
	case TypeClip:
		u.Error = dl.downloadClip(ctx, u)
	case TypeLivestream:
		u.Error = dl.recordLivestream(ctx, u)
	}

	u.notifyDone()

	return u.Error
}

func (dl *Downloader) fetchTitle(ctx context.Context, u *Unit) (string, error) {
	switch u.Type {
	case TypeClip:
		return dl.gql.ClipTitle(ctx, u.ID)
	case TypeVOD:
		return dl.gql.VideoTitle(ctx, u.ID)
	case TypeLivestream:
		return dl.gql.StreamTitle(ctx, u.ID)
	}
	return "", fmt.Errorf("failed to fetch title for unit: %s", u.UUID.String())
}

func (dl *Downloader) openFile(ctx context.Context, u *Unit) error {
	if u.w != nil {
		return nil
	}
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

	f, err := fileutil.Open(u.dir, u.filename, u.ext)
	if err != nil {
		return err
	}
	u.w = f

	return nil
}

func (dl *Downloader) downloadBytes(ctx context.Context, u *Unit, b []byte) error {
	if err := dl.openFile(ctx, u); err != nil {
		return err
	}

	written, err := u.w.Write(b)
	if err != nil {
		return err
	}
	u.notifyProgress(int64(written))

	return nil
}

func withSegmentSuffix(segURL, replacer string) string {
	parts := strings.Split(segURL, "/")
	filename := parts[len(parts)-1]

	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	if i := strings.IndexByte(name, '-'); i >= 0 {
		name = name[:i]
	}

	parts[len(parts)-1] = name + replacer + ext
	return strings.Join(parts, "/")
}

func segmentSuffix(segURL string) string {
	_, filename := filepath.Split(segURL)
	name, found := strings.CutSuffix(filename, path.Ext(filename))
	if !found {
		panic("assert: must find the extension")
	}
	parts := strings.Split(name, "-")
	if len(parts) == 1 {
		return ""
	}
	return "-" + parts[1]
}

func (dl *Downloader) fetchSegment(ctx context.Context, u *Unit, segURL string) (io.ReadCloser, error) {
	candidates := make([]string, 0)

	recovering := u.unmuteFailStreak.Load() <= maxUnmuteFailStreak && segmentSuffix(segURL) == "-muted"
	if recovering {
		candidates = append(candidates, withSegmentSuffix(segURL, "-unmuted"))
		candidates = append(candidates, withSegmentSuffix(segURL, ""))
	}
	candidates = append(candidates, segURL)

	for i, candidate := range candidates {
		resp, err := httputil.Do(ctx, dl.http, candidate, http.MethodGet, nil, nil)
		if err != nil {
			return nil, err
		}

		code := resp.StatusCode

		if code < 200 || code >= 300 {
			if code == http.StatusForbidden {
				resp.Body.Close()
				continue
			}
			b, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read the error response: %v", err)
			}
			return nil, fmt.Errorf("failed to fetch segment - invalid status code %d: response: %s", code, string(b))
		}

		if recovering {
			if i == len(candidates)-1 {
				u.unmuteFailStreak.Add(1)
			} else {
				u.unmuteFailStreak.Store(0)
			}
		}

		return resp.Body, nil
	}

	return nil, fmt.Errorf("failed to fetch the segment: url=%s", segURL)
}

func (dl *Downloader) fetchDownload(ctx context.Context, u *Unit, segURL string) error {
	r, err := dl.fetchSegment(ctx, u, segURL)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := dl.openFile(ctx, u); err != nil {
		return err
	}

	written, err := io.Copy(u.w, r)
	if err != nil {
		return err
	}

	u.notifyProgress(written)

	return nil
}
