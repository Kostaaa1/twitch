package helix

import (
	"context"
	"net/http"
	"net/url"
)

type Channel struct {
	ID                          string   `json:"id"`
	BroadcasterID               string   `json:"broadcaster_id"`
	BroadcasterLogin            string   `json:"broadcaster_login"`
	BroadcasterName             string   `json:"broadcaster_name"`
	BroadcasterLanguage         string   `json:"broadcaster_language"`
	GameID                      string   `json:"game_id"`
	GameName                    string   `json:"game_name"`
	Title                       string   `json:"title"`
	Delay                       int      `json:"delay"`
	Tags                        []string `json:"tags"`
	ContentClassificationLabels []string `json:"content_classification_labels"`
	IsBrandedContent            bool     `json:"is_branded_content"`
}

type channels struct {
	c      *Client
	url    *url.URL
	values url.Values
}

func (c *channels) BroadcasterID(bid string) *channels {
	c.values.Add("broadcaster_id", bid)
	return c
}

func (s *channels) Run(ctx context.Context) (*helixEnvelope[Channel], error) {
	s.url.RawQuery = s.values.Encode()
	var body helixEnvelope[Channel]
	if err := s.c.Request(ctx, s.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Client) Channels() *channels {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/channels")
	return &channels{
		c:      c,
		url:    parsed,
		values: url.Values{},
	}
}
