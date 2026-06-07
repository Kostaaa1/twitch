package twitch

import (
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
)

type Client struct {
	Helix *helix.Client
	Gql   *gql.Client
}

type clientOpts func(*Client)

func NewClient(opts ...clientOpts) *Client {
	c := &Client{
		Helix: helix.New(),
		Gql:   gql.New(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithHTTPClient(hc *http.Client) clientOpts {
	return func(c *Client) {
		c.Helix.SetHTTPClient(hc)
		c.Gql.SetHTTPClient(hc)
	}
}

// func WithOAuthCreds(creds *helix.OAuthCreds) clientOpts {
// 	return func(c *Client) {
// 		c.Helix.OAuthCreds = creds
// 	}
// }
