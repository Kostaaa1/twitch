package twitch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type helixEnvelope[T any] struct {
	Data []T `json:"data"`
}

// It has retry mechanism for 401 Unauthorized responses, it will attempt to refresh the access token and retry the request up to Client.retryCount times.
func (tw *Client) HelixRequest(
	ctx context.Context,
	url string,
	httpMethod string,
	body io.Reader,
	src interface{},
) error {
	return errors.New("helix request not implemented")

	// if err := tw.ensureValidCreds(ctx); err != nil {
	// 	return err
	// }

	// retryCount := 0
	// var errResp ErrorResponse

	// decodeErr := func(r io.Reader) error {
	// 	return json.NewDecoder(r).Decode(&errResp)
	// }

	// for {
	// 	req, err := http.NewRequestWithContext(ctx, httpMethod, url, body)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to create request: %w", err)
	// 	}
	// 	req.Header.Set("Client-Id", tw.oauthCreds.ClientID)
	// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tw.oauthCreds.AccessToken))
	// 	req.Header.Set("Content-Type", "application/json")

	// 	resp, err := tw.http.Do(req)
	// 	if err != nil {
	// 		return fmt.Errorf("request failed: %v", err)
	// 	}
	// 	defer resp.Body.Close()

	// 	if resp.StatusCode == http.StatusUnauthorized {
	// 		if retryCount >= tw.retryCount {
	// 			return fmt.Errorf("max retries (%d) reached for unauthorized requests", tw.retryCount)
	// 		}
	// 		if err := decodeErr(resp.Body); err != nil {
	// 			return fmt.Errorf("failed to decode error response: %v", err)
	// 		}
	// 		if err := tw.FetchAccesToken(ctx); err != nil {
	// 			return fmt.Errorf("failed to refresh access token: %v", err)
	// 		}
	// 		retryCount++
	// 		continue
	// 	}

	// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
	// 		if err := decodeErr(resp.Body); err != nil {
	// 			return fmt.Errorf("failed to decode error response: %v", err)
	// 		}
	// 		return fmt.Errorf("invalid status code: message=%s | code=%d", errResp.Message, resp.StatusCode)
	// 	}

	// 	if resp.ContentLength == 0 || resp.StatusCode == http.StatusNoContent {
	// 		return nil
	// 	}

	// 	if err := json.NewDecoder(resp.Body).Decode(&src); err != nil {
	// 		return fmt.Errorf("failed to decode response: %v", err)
	// 	}

	// 	return nil
	// }
}

func (tw *Client) UserByChannelName(ctx context.Context, channelName string) (*User, error) {
	url := fmt.Sprintf("%s/users", helixURL)
	if channelName != "" {
		url += "?login=" + channelName
	}

	var body helixEnvelope[User]
	if err := tw.HelixRequest(ctx, url, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get user data for: %s", channelName)
}

func (tw *Client) UserByID(ctx context.Context, id string) (*User, error) {
	url := fmt.Sprintf("%s/users?id=%s", helixURL, id)

	var body helixEnvelope[User]
	if err := tw.HelixRequest(ctx, url, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get user data by id: %s", id)
}

func (tw *Client) ChannelInfo(ctx context.Context, broadcasterID string) (*Channel, error) {
	u := fmt.Sprintf("%s/channels?broadcaster_id=%s", helixURL, broadcasterID)

	var body helixEnvelope[Channel]
	if err := tw.HelixRequest(ctx, u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get the channel info for: %s", broadcasterID)
}

func (tw *Client) FollowedStreams(ctx context.Context, id string) (*[]Stream, error) {
	u := fmt.Sprintf("%s/streams/followed?user_id=%s", helixURL, id)

	var body helixEnvelope[[]Stream]
	if err := tw.HelixRequest(ctx, u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	if len(body.Data) > 0 {
		return &body.Data[0], nil
	}

	return nil, fmt.Errorf("failed to get followed streams by user id: %s", id)
}

func (tw *Client) Stream(ctx context.Context, userId string) (*[]Stream, error) {
	u := fmt.Sprintf("%s/streams?user_id=%s", helixURL, userId)
	var body []Stream
	if err := tw.HelixRequest(ctx, u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}
