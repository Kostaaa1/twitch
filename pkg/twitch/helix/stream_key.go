package helix

import (
	"context"
	"net/http"
	"net/url"
)

type streamKey struct {
	c      *Client
	url    *url.URL
	values url.Values
}

type StreamKey struct {
	StreamKey string `json:"stream_key"`
}

func (s *streamKey) Run(ctx context.Context) ([]StreamKey, error) {
	s.url.RawQuery = s.values.Encode()
	var body helixEnvelope[StreamKey]
	if err := s.c.Request(ctx, s.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *Client) StreamKey(broadcasterID string) *streamKey {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/streams/key")
	v := url.Values{}
	v.Set("broadcaster_id", broadcasterID)
	return &streamKey{
		c:      c,
		url:    parsed,
		values: v,
	}
}
