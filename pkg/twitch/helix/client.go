package helix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	helixURL = "https://api.twitch.tv/helix"
	// usherURL = "https://usher.ttvnw.net"
)

type Client struct {
	http       *http.Client
	OAuthCreds *OAuthCreds
	eventsub   *Eventsub
}

func New() *Client {
	return &Client{http: http.DefaultClient}
}

func (c *Client) SetHTTPClient(hc *http.Client) { c.http = hc }

type clientOpts func(*Client)

func WithOAuthCreds(creds *OAuthCreds) clientOpts {
	return func(c *Client) {
		c.OAuthCreds = creds
	}
}

func WithEventsub() clientOpts {
	return func(c *Client) {
		c.eventsub = NewEventsub(c)
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

func (h *Client) Bearer() string {
	return fmt.Sprintf("Bearer %s", h.OAuthCreds.UserToken.AccessToken)
}

func (h *Client) Request(
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
	src interface{},
) error {
	if src == nil {
		return errors.New("src not defined")
	}

	if err := h.ensureValidCreds(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", h.OAuthCreds.ClientID)
	req.Header.Set("Authorization", h.Bearer())

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

	if err := json.NewDecoder(resp.Body).Decode(&src); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	return nil
}
