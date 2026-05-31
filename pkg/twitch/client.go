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
	http := http.DefaultClient

	c := &Client{
		Helix: helix.New(http),
		Gql:   &gql.Client{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithOAuthCreds(creds *helix.OAuthCreds) clientOpts {
	return func(c *Client) {
		c.Helix.OAuthCreds = creds
	}
}
