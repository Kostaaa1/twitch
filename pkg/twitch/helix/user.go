package helix

import (
	"context"
	"net/http"
	"net/url"
)

type User struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	Email           string `json:"email"`
	CreatedAt       string `json:"created_at"`
}

type user struct {
	c      *Client
	url    *url.URL
	values url.Values
}

func (c *user) UserID(id string) *user {
	c.values.Add("id", id)
	return c
}

func (c *user) UserLogin(login string) *user {
	c.values.Add("login", login)
	return c
}

func (s *user) Run(ctx context.Context) (*helixEnvelope[User], error) {
	s.url.RawQuery = s.values.Encode()
	var body helixEnvelope[User]
	if err := s.c.Request(ctx, s.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Client) Users() *user {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/users")
	return &user{
		c:      c,
		url:    parsed,
		values: url.Values{},
	}
}
