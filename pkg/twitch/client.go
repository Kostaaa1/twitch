package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		Helix: &helix.Client{},
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

func fetchWithDecode(
	ctx context.Context,
	httpClient *http.Client,
	url string,
	method string,
	body io.Reader,
	dst any,
	h http.Header,
) error {
	if dst == nil {
		return errors.New("dst cannot be nil")
	}
	if url == "" {
		return errors.New("failed to fetch: missing url")
	}
	if method == "" {
		return errors.New("failed to fetch: missing method")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create the request: url=%s err=%v", url, err)
	}
	req.Header = h.Clone()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read the error response: %v", err)
		}
		return fmt.Errorf("invalid status %d: %s", resp.StatusCode, string(b))
	}

	if resp.Body != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return err
		}
	}

	return nil
}

func (tw *Client) request(
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
	h http.Header,
) (*http.Response, error) {
	if url == "" {
		return nil, errors.New("failed to fetch: missing url")
	}
	if method == "" {
		return nil, errors.New("failed to fetch: missing method")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create the request: url=%s err=%v", url, err)
	}
	req.Header = h.Clone()

	resp, err := tw.http.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
