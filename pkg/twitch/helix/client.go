package helix

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Kostaaa1/twitch/internal/httputil"
)

type Client struct {
	http       *http.Client
	OAuthCreds *OAuthCreds
}

func New(httpClient *http.Client, opts ...clientOpts) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	c := &Client{http: httpClient}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type clientOpts func(*Client)

func WithOAuthCreds(creds *OAuthCreds) clientOpts {
	return func(c *Client) {
		c.OAuthCreds = creds
	}
}

type HelixErrResponse struct {
	Err     string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (e HelixErrResponse) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.Err, e.Status, e.Message)
}

type helixEnvelope[T any] struct {
	Data []T `json:"data"`
}

type helixPaginatedEnvelope[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

func (c *Client) bearerUserToken() string {
	return fmt.Sprintf("Bearer %s", c.OAuthCreds.UserToken.AccessToken)
}

func (c *Client) bearerAppToken() string {
	return fmt.Sprintf("Bearer %s", c.OAuthCreds.AppToken.AccessToken)
}

func (c *Client) RequestWithAppToken(
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
	dst interface{},
) error {
	if c.OAuthCreds.ClientID == "" {
		return ErrMissingClientID
	}

	if c.OAuthCreds.AppToken.AccessToken == "" || c.OAuthCreds.AppToken.Expired() {
		if err := c.AppAccessToken(ctx); err != nil {
			return err
		}
	}

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Client-Id", c.OAuthCreds.ClientID)
	h.Set("Authorization", c.bearerAppToken())

	return httputil.DoJSON(
		ctx,
		c.http,
		url,
		method,
		body,
		dst,
		h,
	)
}

func (c *Client) RequestWithAccessToken(
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
	dst interface{},
) error {
	if c.OAuthCreds.ClientID == "" {
		return ErrMissingClientID
	}

	if c.OAuthCreds.UserToken.Expired() {
		if err := c.RefreshAccessToken(ctx); err != nil {
			return err
		}
	}

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Client-Id", c.OAuthCreds.ClientID)
	h.Set("Authorization", c.bearerUserToken())

	return httputil.DoJSON(
		ctx,
		c.http,
		url,
		method,
		body,
		dst,
		h,
	)
}
