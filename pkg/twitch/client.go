package twitch

import (
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
)

type Client struct {
	// http  *http.Client
	Helix *helix.Client
	Gql   *gql.Client
}

type clientOpts func(*Client)

func NewClient(opts ...clientOpts) *Client {
	c := &Client{
		Helix: helix.New(),
		Gql:   &gql.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithHttpClient(httpClient *http.Client) clientOpts {
	return func(c *Client) {
	}
}

func WithOAuthCreds(creds *helix.OAuthCreds) clientOpts {
	return func(c *Client) {
		c.Helix.oauthCreds = creds
	}
}
