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

func (dl *Downloader) fetchMediaPlaylist(ctx context.Context, url string) (*m3u8.MediaPlaylist, error) {
	resp, err := httputil.Do(ctx, dl.http, url, http.MethodGet, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	playlist, err := m3u8.ParseMediaPlaylist(resp.Body, url)
	if err != nil {
		return nil, err
	}

	return playlist, nil
}

func (dl *Downloader) mediaPlaylistForUnit(ctx context.Context, unit *Unit) (*m3u8.MediaPlaylist, error) {
	master, err := dl.MasterPlaylistVOD(ctx, unit.ID)
	if err != nil {
		return nil, err
	}

	variant, err := master.VariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return nil, err
	}

	playlist, err := dl.fetchMediaPlaylist(ctx, variant.URL)
	if err != nil {
		return nil, err
	}

	if unit.Start > 0 || unit.End > 0 {
		playlist.Truncate(unit.Start, unit.End)
	}

	return playlist, nil
}

func parsePreviewURL(previewURL string) (string, string, error) {
	u, err := url.Parse(previewURL)
	if err != nil {
		return "", "", err
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 1 {
		return "", "", errors.New("")
	}

	subdomain := strings.Split(u.Hostname(), ".")[0]
	id := parts[0]

	return subdomain, id, nil
}

// TODO: whole process of creating mock master playlists could be avoided
func (dl *Downloader) mockMasterPlaylist(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	bt, previewURL, err := dl.gql.SeekPreviewsURL(ctx, vodID)
	if err != nil {
		return nil, err
	}

	if previewURL == "" {
		return nil, fmt.Errorf("failed to acquire previewURL for video: %s", vodID)
	}

	subdomain, id, err := parsePreviewURL(previewURL)
	if err != nil {
		return nil, err
	}

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
			listURL = fmt.Sprintf("https://%s.cloudfront.net/%s/%s/index-dvr.m3u8", subdomain, id, key)
		default:
			return nil, errors.New("unsupported broadcast type")
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

	return &master, nil
}

func (dl *Downloader) MasterPlaylistVOD(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	tok, err := dl.gql.VideoPlaybackAccessToken(ctx, vodID)
	if err != nil {
		return nil, err
	}

	m3u8url := fmt.Sprintf("%s/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true",
		gql.UsherURL,
		vodID,
		tok.Value,
		tok.Signature,
	)

	b, code, err := httputil.DoBytes(ctx, dl.http, m3u8url, http.MethodGet, nil, nil)
	if code == http.StatusForbidden {
		return dl.mockMasterPlaylist(ctx, vodID)
	}

	if err != nil {
		return nil, err
	}

	return m3u8.Master(b), nil
}

func stripSegmentURLType(url string) string {
	// TODO: does not work for init-0.mp4
	// strip '-unmuted', '-muted'
	// id := strings.LastIndex(url, "-")
	// if id == -1 {
	// 	return url
	// }
	// return fmt.Sprintf("%s%s", url[:id], filepath.Ext(url))

	if strings.Contains(url, "-unmuted") {
		return strings.Replace(url, "-unmuted", "", 1)
	}
	if strings.Contains(url, "-muted") {
		return strings.Replace(url, "-muted", "", 1)
	}
	return url
}

func buildSegURL(playlistURL, path string) string {
	lastIndex := strings.LastIndex(playlistURL, "/")
	return fmt.Sprintf("%s/%s", playlistURL[:lastIndex], path)
}

func (dl *Downloader) downloadVOD(ctx context.Context, unit *Unit) error {
	list, err := dl.mediaPlaylistForUnit(ctx, unit)
	if err != nil {
		return err
	}

	if list.Map != nil && list.Map.URI != "" {
		if err := dl.segmentFetchDownload(ctx, unit, buildSegURL(list.URL, list.Map.URI)); err != nil {
			return err
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	currentChunk := atomic.Uint32{}

	depth := make(chan struct{}, unit.readAheadDepth)
	workerCount := 4

	for i := 0; i < workerCount; i++ {
		g.Go(func() error {
			for {
				chunkInx := int(currentChunk.Add(1) - 1)
				if chunkInx >= len(list.Segments) {
					return nil
				}

				select {
				case depth <- struct{}{}:
				case <-ctx.Done():
					return ctx.Err()
				}

				seg := list.Segments[chunkInx]
				segURL := buildSegURL(list.URL, seg.URI)

				body, err := dl.fetchSegment(ctx, unit, segURL)
				if err != nil {
					return err
				}

				b, err := io.ReadAll(body)
				body.Close()
				if err != nil {
					return err
				}

				seg.Data <- b
				close(seg.Data)
			}
		})
	}

	g.Go(func() error {
		for i := 0; i < len(list.Segments); i++ {
			select {
			case <-depth:
			case <-ctx.Done():
				return ctx.Err()
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case b := <-list.Segments[i].Data:
				if err := dl.downloadBytes(unit, b); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return g.Wait()
}
