package helix

import (
	"context"
	"net/http"
	"net/url"
)

// Get Followed Streams
// Gets the list of broadcasters that the user follows and that are streaming live.

// Authorization
// Requires a user access token that includes the user:read:follows scope.

// URL
// GET https://api.twitch.tv/helix/streams/followed

type Game struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BoxArtURL string `json:"box_art_url"`
	IgdbID    string `json:"igdb_id"`
}

type games struct {
	c      *Client
	url    *url.URL
	values url.Values
}

func (s *games) Name(name string) *games {
	s.values.Add("name", name)
	return s
}

func (s *games) ID(id string) *games {
	s.values.Add("id", id)
	return s
}

func (s *games) IgdbID(id string) *games {
	s.values.Add("igdb_id", id)
	return s
}

func (s *games) Run(ctx context.Context) (*helixEnvelope[Game], error) {
	s.url.RawQuery = s.values.Encode()
	var body helixEnvelope[Game]
	if err := s.c.Request(ctx, s.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Client) Games() *games {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/games")
	return &games{
		c:      c,
		url:    parsed,
		values: url.Values{},
	}
}
