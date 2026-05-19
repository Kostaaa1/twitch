package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	gqlURL      = "https://gql.twitch.tv/gql"
	gqlClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	usherURL    = "https://usher.ttvnw.net"
	helixURL    = "https://api.twitch.tv/helix"
	oauthURL    = "https://id.twitch.tv/oauth2"
)

type Client struct {
	http       *http.Client
	retryCount int
	Helix      *Helix
}

type clientOpts func(*Client)

func NewClient(opts ...clientOpts) *Client {
	c := &Client{
		http:       http.DefaultClient,
		retryCount: 3,
		Helix:      &Helix{},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Helix.http = c.http

	return c
}

func WithOAuthCreds(creds *OAuthCreds) clientOpts {
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

func sendGqlLoadAndDecode[T any](
	ctx context.Context,
	c *http.Client,
	dst *T,
	gqlLoad string,
	a ...any,
) error {
	type response struct {
		Data       T `json:"data"`
		Extensions struct {
			DurationMilliseconds int    `json:"durationMilliseconds"`
			OperationName        string `json:"operationName"`
			RequestID            string `json:"requestID"`
		} `json:"extensions"`
	}

	var resp response

	var r io.Reader
	if len(a) > 0 {
		r = strings.NewReader(fmt.Sprintf(gqlLoad, a...))
	} else {
		r = strings.NewReader(gqlLoad)
	}

	h := http.Header{}
	h.Set("Client-Id", gqlClientID)
	h.Set("Content-Type", "application/json")

	if err := fetchWithDecode(ctx, c, gqlURL, http.MethodPost, r, &resp, h); err != nil {
		return err
	}

	// must be pointer
	*dst = resp.Data

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
