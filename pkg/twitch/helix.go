package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Stream struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	UserLogin    string        `json:"user_login"`
	UserName     string        `json:"user_name"`
	GameID       string        `json:"game_id"`
	GameName     string        `json:"game_name"`
	Type         string        `json:"type"`
	Title        string        `json:"title"`
	ViewerCount  int           `json:"viewer_count"`
	StartedAt    time.Time     `json:"started_at"`
	Language     string        `json:"language"`
	ThumbnailURL string        `json:"thumbnail_url"`
	TagIds       []interface{} `json:"tag_ids"`
	Tags         []string      `json:"tags"`
	IsMature     bool          `json:"is_mature"`
}

type Channel struct {
	BroadcasterID               string   `json:"broadcaster_id"`
	BroadcasterLogin            string   `json:"broadcaster_login"`
	BroadcasterName             string   `json:"broadcaster_name"`
	BroadcasterLanguage         string   `json:"broadcaster_language"`
	GameID                      string   `json:"game_id"`
	GameName                    string   `json:"game_name"`
	Title                       string   `json:"title"`
	Delay                       int      `json:"delay"`
	Tags                        []string `json:"tags"`
	ContentClassificationLabels []string `json:"content_classification_labels"`
	IsBrandedContent            bool     `json:"is_branded_content"`
}

type helixEnvelope[T any] struct {
	Data []T `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// It has retry mechanism for 401 Unauthorized responses, it will attempt to refresh the access token and retry the request up to Client.retryCount times.
func (tw *Client) HelixRequest(
	ctx context.Context,
	url string,
	httpMethod string,
	body io.Reader,
	src interface{},
) error {
	retryCount := 0
	var errResp ErrorResponse

	decodeErr := func(r io.Reader) error {
		if err := json.NewDecoder(r).Decode(&errResp); err != nil {
			return err
		}
		return nil
	}

	for {
		req, err := http.NewRequestWithContext(ctx, httpMethod, url, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Client-Id", tw.creds.ClientID)
		req.Header.Set("Authorization", tw.GetBearerToken())
		req.Header.Set("Content-Type", "application/json")

		resp, err := tw.http.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			if retryCount >= tw.retryCount {
				return fmt.Errorf("max retries (%d) reached for unauthorized requests", tw.retryCount)
			}

			if err := decodeErr(resp.Body); err != nil {
				return fmt.Errorf("failed to decode error response: %v", err)
			}

			if err := tw.RefetchAccesToken(); err != nil {
				return fmt.Errorf("failed to refresh access token: %v", err)
			}

			retryCount++
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if err := decodeErr(resp.Body); err != nil {
				return fmt.Errorf("failed to decode error response: %v", err)
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

type User struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	Email           string `json:"email"`
	CreatedAt       string `json:"created_at"`
}

// Returns user data by channel name. If channel name is empty, it returns data for authenticated user
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

// remove decapi - handle this with graphql
func (tw *Client) IsChannelLive(ctx context.Context, channelName string) (bool, error) {
	data, err := tw.StreamMetadata(ctx, channelName)
	if err != nil {
		return false, fmt.Errorf("failed to get the stream metadata for user: %s. error: %v", channelName, err)
	}

	return len(data.Stream.ID) > 0, nil
}
