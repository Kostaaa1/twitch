package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	http       *http.Client
	oauthCreds *OAuthCreds
	retryCount int
}

const (
	gqlURL      = "https://gql.twitch.tv/gql"
	gqlClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	usherURL    = "https://usher.ttvnw.net"
	helixURL    = "https://api.twitch.tv/helix"
	oauthURL    = "https://id.twitch.tv/oauth2"
)

func NewClient(c *OAuthCreds) *Client {
	return &Client{
		oauthCreds: c,
		http:       http.DefaultClient,
		retryCount: 3,
	}
}

func (tw *Client) fetchWithDecode(
	ctx context.Context,
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

	resp, err := tw.http.Do(req)
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

func (tw *Client) sendGqlLoadAndDecode(ctx context.Context, r io.Reader, dst any) error {
	h := http.Header{}
	h.Set("Client-Id", gqlClientID)
	h.Set("Content-Type", "application/json")
	return tw.fetchWithDecode(ctx, gqlURL, http.MethodPost, r, dst, h)
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
