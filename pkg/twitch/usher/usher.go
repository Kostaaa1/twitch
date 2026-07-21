package usher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kostaaa1/twitch/internal/downloader/m3u8"
	"github.com/Kostaaa1/twitch/internal/httputil"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
)

type Client struct {
	gql  *gql.Client
	http *http.Client
}

func New(gql *gql.Client, http *http.Client) *Client {
	return &Client{gql, http}
}

func (c *Client) MasterPlaylistStream(ctx context.Context, channel string) (*m3u8.MasterPlaylist, error) {
	tok, err := c.gql.StreamPlaybackAccessToken(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestream credentials: %w", err)
	}

	url := fmt.Sprintf(
		"https://usher.ttvnw.net/api/channel/hls/%s.m3u8?token=%s&sig=%s&allow_audio_only=true&allow_source=true",
		channel,
		tok.Value,
		tok.Signature,
	)

	b, _, err := httputil.DoBytes(
		ctx,
		c.http,
		url,
		http.MethodGet,
		nil,
		nil,
	)

	return m3u8.Master(b), nil
}

func (c *Client) MasterPlaylistVideo(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	tok, err := c.gql.VideoPlaybackAccessToken(ctx, vodID)
	if err != nil {
		return nil, err
	}

	// TODO: investigate params
	m3u8url := fmt.Sprintf(
		"https://usher.ttvnw.net/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true",
		vodID,
		tok.Value,
		tok.Signature,
	)

	b, code, err := httputil.DoBytes(ctx, c.http, m3u8url, http.MethodGet, nil, nil)
	if code == http.StatusForbidden {
		return c.mockMasterPlaylist(ctx, vodID)
	}

	if err != nil {
		return nil, err
	}

	return m3u8.Master(b), nil
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

func (c *Client) mockMasterPlaylist(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	bt, previewURL, err := c.gql.SeekPreviewsURL(ctx, vodID)
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
		resp, err := c.http.Get(u)
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
			listURL = fmt.Sprintf(
				"https://%s.cloudfront.net/%s/%s/index-dvr.m3u8",
				subdomain,
				id,
				key,
			)
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
