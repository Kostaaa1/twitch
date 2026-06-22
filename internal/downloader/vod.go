package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/Kostaaa1/twitch/internal/downloader/m3u8"
	"github.com/Kostaaa1/twitch/internal/httputil"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"golang.org/x/sync/errgroup"
)

func (dl *Downloader) mediaPlaylistForUnit(ctx context.Context, unit *Unit) (*m3u8.MediaPlaylist, error) {
	master, err := dl.MasterPlaylistVOD(ctx, unit.ID)
	if err != nil {
		return nil, err
	}

	variant, err := master.VariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, variant.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := dl.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	playlist, err := m3u8.ParseMediaPlaylist(resp.Body, variant.URL)
	if err != nil {
		return nil, err
	}

	if unit.Start > 0 || unit.End > 0 {
		playlist.Truncate(unit.Start, unit.End)
	}

	return playlist, nil
}

func (dl *Downloader) mockMasterPlaylist(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	bt, previewURL, err := dl.twClient.Gql.SeekPreviewsURL(ctx, vodID)
	if err != nil {
		return nil, err
	}

	if previewURL == "" {
		return nil, fmt.Errorf("failed to acquire previewURL for video: %s", vodID)
	}

	u, err := url.Parse(previewURL)
	if err != nil {
		return nil, err
	}

	subdomain := strings.Split(u.Host, ".")[0]
	fmt.Println("Subdomain", subdomain)
	fmt.Println("Preview", previewURL)

	master := m3u8.MasterPlaylist{
		Origin:          "s3",
		B:               false,
		Region:          "EU",
		UserIP:          "127.0.0.1",
		Cluster:         "cloudfront_vod",
		UserCountry:     "BE",
		ManifestCluster: "cloudfront_vod",
	}

	resolutions := map[string]struct {
		Res string
		FPS string
	}{
		"chunked":    {Res: "1920x1080", FPS: "60"},
		"720p60":     {Res: "1280x720", FPS: "60"},
		"720p30":     {Res: "1280x720", FPS: "30"},
		"480p30":     {Res: "854x480", FPS: "30"},
		"360p30":     {Res: "640x360", FPS: "30"},
		"160p30":     {Res: "284x160", FPS: "30"},
		"audio_only": {Res: "audio_only", FPS: ""},
	}

	isQualityValid := func(u string) bool {
		resp, err := dl.http.Get(u)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}

	for key, value := range resolutions {
		var listURL string

		switch bt {
		case "UPLOAD":
		case "HIGHLIGHT":
		case "ARCHIVE":
		}

		if listURL == "" {
			return nil, errors.New("failed to create mock master playlist: missing list url")
		}

		if isQualityValid(listURL) {
			vp := &m3u8.VariantPlaylist{
				URL:        listURL,
				Bandwidth:  "", // ????
				Codecs:     "avc1.64002A,mp4a.40.2",
				Resolution: value.Res,
				FrameRate:  value.FPS,
				Video:      key,
			}
			master.Lists = append(master.Lists, vp)
		}
	}

	for _, list := range master.Lists {
		fmt.Println("List:", list)
	}

	return &master, nil
}

func (dl *Downloader) MasterPlaylistVOD(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	tok, err := dl.twClient.Gql.VideoPlaybackAccessToken(ctx, vodID)
	if err != nil {
		return nil, err
	}

	m3u8url := fmt.Sprintf("%s/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true",
		gql.UsherURL,
		vodID,
		tok.Value,
		tok.Signature,
	)

	b, code, err := httputil.Fetch(ctx, dl.http, m3u8url, http.MethodGet, nil, nil)
	if err != nil {
		return nil, err
	}

	if code == http.StatusForbidden {
		return dl.mockMasterPlaylist(ctx, vodID)
	}

	return m3u8.Master(b), nil
}

func transformForbiddenSegURL(url string) (string, error) {
	switch {
	case strings.Contains(url, "-unmuted"):
		return strings.Replace(url, "-unmuted", "-muted", 1), nil
	case strings.Contains(url, "-muted"):
		return strings.Replace(url, "-muted", "", 1), nil
	}
	return "", fmt.Errorf("forbidden for segment: %s", url)
}

func (dl *Downloader) fetchSegment(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := dl.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusForbidden {
		url, err = transformForbiddenSegURL(url)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		resp, err = dl.http.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp.Body, nil
}

func (dl *Downloader) downloadVOD(ctx context.Context, unit *Unit) error {
	playlist, err := dl.mediaPlaylistForUnit(ctx, unit)
	if err != nil {
		return err
	}

	// TODO: look into this
	workerCount := 4

	g, ctx := errgroup.WithContext(ctx)
	currentChunk := atomic.Uint32{}

	for i := 0; i < workerCount; i++ {
		g.Go(func() error {
			for {
				chunkInx := int(currentChunk.Add(1) - 1)
				if chunkInx >= len(playlist.Segments) {
					return nil
				}

				seg := playlist.Segments[chunkInx]

				if !strings.HasSuffix(seg.URL, ".ts") {
					return errors.New("malformed playlist segment url: does not have .ts extension")
				}

				lastIndex := strings.LastIndex(playlist.URL, "/")
				tsURL := fmt.Sprintf("%s/%s", playlist.URL[:lastIndex], seg.URL)

				body, err := dl.fetchSegment(ctx, tsURL)
				if err != nil {
					return err
				}

				seg.Data <- body
				close(seg.Data)
			}
		})
	}

	g.Go(func() error {
		for i := 0; i < len(playlist.Segments); i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case chunk := <-playlist.Segments[i].Data:
				if err := unit.download(dl, chunk); err != nil {
					return err
				}
			}
		}
		return nil
	})

	return g.Wait()
}
