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
	oauthCreds *OAuthCreds
}

func New(creds *OAuthCreds, opts ...clientOpts) (*Client, error) {
	if creds == nil {
		return nil, ErrMissingOAuthCredentials
	}

	c := &Client{oauthCreds: creds}
	for _, opt := range opts {
		opt(c)
	}

	if c.http == nil {
		c.http = http.DefaultClient
	}

	return c, nil
}

type clientOpts func(*Client)

func WithHTTPClient(httpClient *http.Client) clientOpts {
	return func(c *Client) {
		c.http = httpClient
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
	return fmt.Sprintf("Bearer %s", c.oauthCreds.UserToken.AccessToken)
}

func (c *Client) bearerAppToken() string {
	return fmt.Sprintf("Bearer %s", c.oauthCreds.AppToken.AccessToken)
}

func (c *Client) RequestWithAppToken(
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
	dst interface{},
) error {
	if c.oauthCreds.ClientID == "" {
		return ErrMissingClientID
	}

	if c.oauthCreds.AppToken.AccessToken == "" || c.oauthCreds.AppToken.Expired() {
		if err := c.AppAccessToken(ctx); err != nil {
			return err
		}
	}

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Client-Id", c.oauthCreds.ClientID)
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
	if c.oauthCreds.ClientID == "" {
		return ErrMissingClientID
	}

	if c.oauthCreds.UserToken.Expired() {
		if err := c.RefreshAccessToken(ctx); err != nil {
			return err
		}
	}

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Client-Id", c.oauthCreds.ClientID)
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
