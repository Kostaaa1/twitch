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
	usherURL = "https://usher.ttvnw.net"
)

type Client struct {
	http       *http.Client
	OAuthCreds *OAuthCreds
	eventsub   *Eventsub
}

func New(http *http.Client) *Client {
	return &Client{http: http}
}

type clientOpts func(*Client)

// func WithOAuthCreds(creds *OAuthCreds) clientOpts {
// 	return func(c *Client) {
// 		c.OAuthCreds = creds
// 	}
// }

func WithEventsub() clientOpts {
	return func(c *Client) {
		c.eventsub = NewEventsub(c)
	}
}

type HelixErrResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type helixEnvelope[T any] struct {
	Data []T `json:"data"`
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

	retryCount := 0
	var errResp HelixErrResponse

	for {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Client-Id", h.OAuthCreds.ClientID)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.OAuthCreds.UserToken.AccessToken))
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.http.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			if retryCount >= 3 {
				return fmt.Errorf("max retries (%d) reached for unauthorized requests", 3)
			}
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
				return err
			}
			if err := h.UserTokenWithRefreshToken(ctx); err != nil {
				return fmt.Errorf("failed to refresh access token: %v", err)
			}
			retryCount++
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
				return err
			}
			return fmt.Errorf("invalid status code: message=%s | code=%d", errResp.Message, resp.StatusCode)
		}

		if resp.ContentLength == 0 || resp.StatusCode == http.StatusNoContent {
			return nil
		}

		if err := json.NewDecoder(resp.Body).Decode(&src); err != nil {
			return fmt.Errorf("failed to decode response: %v", err)
		}

		return nil
	}
}
