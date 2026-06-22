package helix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (h *Client) bearerUserToken() string {
	return fmt.Sprintf("Bearer %s", h.OAuthCreds.UserToken.AccessToken)
}

func (h *Client) bearerAppToken() string {
	return fmt.Sprintf("Bearer %s", h.OAuthCreds.AppToken.AccessToken)
}

func (h *Client) RequestWithAppToken(
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
	dst interface{},
) error {
	if h.OAuthCreds.ClientID == "" {
		return ErrMissingClientID
	}

	if h.OAuthCreds.AppToken.AccessToken == "" || h.OAuthCreds.AppToken.Expired() {
		if err := h.AppAccessToken(ctx); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", h.OAuthCreds.ClientID)
	req.Header.Set("Authorization", h.bearerAppToken())

	resp, err := h.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errResp HelixErrResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return err
		}
		return errResp
	}

	if resp.ContentLength == 0 || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(&dst); err != nil {
			return fmt.Errorf("failed to decode response: %v", err)
		}
	}

	return nil
}

func (h *Client) Request(
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
	dst interface{},
) error {
	if h.OAuthCreds.ClientID == "" {
		return ErrMissingClientID
	}

	if h.OAuthCreds.UserToken.Expired() {
		if err := h.RefreshAccessToken(ctx); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", h.OAuthCreds.ClientID)
	req.Header.Set("Authorization", h.bearerUserToken())

	resp, err := h.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errResp HelixErrResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return err
		}
		return errResp
	}

	if resp.ContentLength == 0 || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(&dst); err != nil {
			return fmt.Errorf("failed to decode response: %v", err)
		}
	}

	return nil
}
