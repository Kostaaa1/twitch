package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
)

func (dl *Downloader) MasterPlaylistStream(ctx context.Context, channel string) ([]byte, error) {
	tok, err := dl.twClient.Gql.StreamPlaybackAccessToken(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestream credentials: %w", err)
	}

	url := fmt.Sprintf("%s/api/channel/hls/%s.m3u8?token=%s&sig=%s&allow_audio_only=true&allow_source=true", gql.UsherURL, channel, tok.Value, tok.Signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := dl.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}
