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
	oauthCreds *OAuthCreds
	eventsub   *Eventsub
}

func New() *Client {
	return &Client{}
}

type clientOpts func(*Client)

func WithOAuthCreds(creds *OAuthCreds) clientOpts {
	return func(c *Client) {
		c.oauthCreds = creds
	}
}

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

func (h *Client) Request(
	ctx context.Context,
	url string,
	httpMethod string,
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
		req, err := http.NewRequestWithContext(ctx, httpMethod, url, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Client-Id", h.oauthCreds.ClientID)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.oauthCreds.UserToken.AccessToken))
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

type helixEnvelope[T any] struct {
	Data []T `json:"data"`
}

func (h *Client) UserByChannelName(ctx context.Context, channelName string) (*User, error) {
	url := fmt.Sprintf("%s/users", helixURL)
	if channelName != "" {
		url += "?login=" + channelName
	}

	var body helixEnvelope[User]
	if err := h.Request(ctx, url, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get user data for: %s", channelName)
}

func (h *Client) UserByID(ctx context.Context, id string) (*User, error) {
	url := fmt.Sprintf("%s/users?id=%s", helixURL, id)

	var body helixEnvelope[User]
	if err := h.Request(ctx, url, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get user data by id: %s", id)
}

func (h *Client) ChannelInfo(ctx context.Context, broadcasterID string) (*Channel, error) {
	u := fmt.Sprintf("%s/channels?broadcaster_id=%s", helixURL, broadcasterID)

	var body helixEnvelope[Channel]
	if err := h.Request(ctx, u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get the channel info for: %s", broadcasterID)
}

func (h *Client) FollowedStreams(ctx context.Context, id string) (*[]Stream, error) {
	u := fmt.Sprintf("%s/streams/followed?user_id=%s", helixURL, id)

	var body helixEnvelope[[]Stream]
	if err := h.Request(ctx, u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get followed streams by user id: %s", id)
}

func (h *Client) Stream(ctx context.Context, userId string) (*[]Stream, error) {
	u := fmt.Sprintf("%s/streams?user_id=%s", helixURL, userId)
	var body []Stream
	if err := h.Request(ctx, u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}
