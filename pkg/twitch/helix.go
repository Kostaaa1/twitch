package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

type Streams struct {
	Data       []Stream `json:"data"`
	Pagination struct {
	} `json:"pagination"`
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

type helixEnvelope[T any] struct {
	Data []T `json:"data"`
}

func (tw *Client) HelixRequest(
	url string,
	httpMethod string,
	body io.Reader,
	src interface{},
) error {
	var retryCount int

	for {
		req, err := http.NewRequest(httpMethod, url, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Client-Id", tw.creds.ClientID)
		req.Header.Set("Authorization", tw.GetBearerToken())
		req.Header.Set("Content-Type", "application/json")

		resp, err := tw.httpClient.Do(req)
		if err != nil {
			if resp != nil && resp.Body != nil {
				test, _ := io.ReadAll(resp.Body)
				fmt.Println("ERROR: tes: ", test)
			}
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			if retryCount >= 3 {
				return fmt.Errorf("max retries (%d) reached for unauthorized requests", 3)
			}
			if err := tw.RefetchAccesToken(); err != nil {
				return fmt.Errorf("failed to refresh access token: %w", err)
			}
			retryCount++
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("invalid status code: url=%s | code=%d", url, resp.StatusCode)
		}

		if resp.ContentLength == 0 || resp.StatusCode == http.StatusNoContent {
			return errors.New("response content length is 0")
		}

		if err := json.NewDecoder(resp.Body).Decode(&src); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		return nil
	}
}

// if id and login are nil, your User data will be returned
func (tw *Client) User(id, loginName *string) (*User, error) {
	queryParams := []string{}
	if id != nil {
		queryParams = append(queryParams, fmt.Sprintf("id=%s", *id))
	}
	if loginName != nil {
		queryParams = append(queryParams, fmt.Sprintf("login=%s", *loginName))
	}

	url := fmt.Sprintf("%s/users", helixURL)
	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	var body helixEnvelope[User]
	if err := tw.HelixRequest(url, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}

	return &body.Data[0], nil
}

func (tw *Client) GetChannelInfo(broadcasterID string) (*Channel, error) {
	u := fmt.Sprintf("%s/channels?broadcaster_id=%s", helixURL, broadcasterID)
	var body helixEnvelope[Channel]
	if err := tw.HelixRequest(u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body.Data[0], nil
}

func (tw *Client) GetFollowedStreams(id string) (*Streams, error) {
	u := fmt.Sprintf("%s/streams/followed?user_id=%s", helixURL, id)
	var body helixEnvelope[Streams]
	if err := tw.HelixRequest(u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body.Data[0], nil
}

func (tw *Client) GetStream(userId string) (*Streams, error) {
	u := fmt.Sprintf("%s/streams?user_id=%s", helixURL, userId)
	var body Streams
	if err := tw.HelixRequest(u, http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

// change this, use helix for this
func (tw *Client) IsChannelLive(channelName string) (bool, error) {
	u := fmt.Sprintf("%s/%s", "https://decapi.me/twitch/uptime", channelName)
	b, err := tw.fetch(u)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(string(b), "[Error from Twitch Client]") {
		return false, fmt.Errorf("[Error from Twitch Client]")
	}
	return !strings.Contains(string(b), "offline"), nil
}
