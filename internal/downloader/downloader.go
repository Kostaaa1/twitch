package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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

// func (dl *Downloader) fetchMetadata(ctx context.Context, u *Unit) error {
// 	switch u.Type {
// 	case TypeClip:
// 		data, err := dl.gql.ClipMetadata(ctx, u.ID)
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Println("CLIP DATA", data)
// 		u.metadata = &metadata{
// 			id:          data.ID,
// 			title:       data.Title,
// 			date:        data.CreatedAt,
// 			clipCreator: &data.Curator.DisplayName,
// 		}
// 	case TypeVOD:
// 		data, err := dl.gql.VideoMetadata(ctx, u.ID)
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Println("VOD DATA", data)
// 		u.metadata = &metadata{
// 			id:    data.Video.ID,
// 			title: data.Video.Title,
// 			date:  data.Video.CreatedAt,
// 		}
// 	case TypeLivestream:
// 		data, err := dl.gql.StreamMetadata(ctx, u.ID)
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Println("STREAM DATA")
// 		u.metadata = &metadata{
// 			id:    data.User.Stream.ID,
// 			title: data.User.BroadcastSettings.Title,
// 			date:  data.User.Stream.CreatedAt,
// 		}
// 	}
// 	return nil
// }

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
	body, err := dl.fetchSegment(ctx, u, segURL)
	if err != nil {
		return err
	}
	return dl.download(ctx, u, body)
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

	recovering := u.unmuteFailStreak.Load() <= u.maxUnmuteFailStreak && segmentSuffix(segURL) == "-muted"
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

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if resp.StatusCode == http.StatusForbidden {
				resp.Body.Close()
				continue
			}

			b, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read the error response: %v", err)
			}
			return nil, fmt.Errorf(
				"failed to fetch segment - invalid status code %d: response: %s", resp.StatusCode, string(b),
			)
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
