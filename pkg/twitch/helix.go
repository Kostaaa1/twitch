package twitch

import (
	"encoding/json"
	"fmt"
	"io"
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

type ChannelData struct {
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

type UserData struct {
	ID              string    `json:"id"`
	Login           string    `json:"login"`
	DisplayName     string    `json:"display_name"`
	Type            string    `json:"type"`
	BroadcasterType string    `json:"broadcaster_type"`
	Description     string    `json:"description"`
	ProfileImageURL string    `json:"profile_image_url"`
	OfflineImageURL string    `json:"offline_image_url"`
	ViewCount       int       `json:"view_count"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
}

func (api *API) GetUserInfo(loginName string) (*UserData, error) {
	u := fmt.Sprintf("%s/users?login=%s", api.helixURL, loginName)
	req, err := api.NewGetRequest(u)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-Id", api.config.Creds.ClientID)
	req.Header.Set("Authorization", api.GetToken())
	resp, err := api.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type data struct {
		Data []UserData `json:"data"`
	}
	var user data
	if err := json.Unmarshal(b, &user); err != nil {
		return nil, err
	}
	if len(user.Data) == 0 {
		return nil, fmt.Errorf("the channel %s does not exist", loginName)
	}
	return &user.Data[0], nil
}

func (api *API) GetChannelInfo(broadcasterID string) (*ChannelData, error) {
	u := fmt.Sprintf("%s/channels?broadcaster_id=%s", api.helixURL, broadcasterID)
	req, err := api.NewGetRequest(u)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-Id", api.config.Creds.ClientID)
	req.Header.Set("Authorization", api.GetToken())
	resp, err := api.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type data struct {
		Data []ChannelData `json:"data"`
	}
	var channel data
	if err := json.Unmarshal(b, &channel); err != nil {
		return nil, err
	}
	return &channel.Data[0], nil
}

func (api *API) GetFollowedStreams(id string) (*Streams, error) {
	u := fmt.Sprintf("%s/streams/followed?user_id=%s", api.helixURL, id)
	req, err := api.NewGetRequest(u)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-Id", api.config.Creds.ClientID)
	req.Header.Set("Authorization", api.GetToken())
	resp, err := api.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var streams Streams
	if err := json.Unmarshal(b, &streams); err != nil {
		return nil, err
	}
	return &streams, nil
}

func (api *API) GetStream(userId string) (*Streams, error) {
	u := fmt.Sprintf("%s/streams?user_id=%s", api.helixURL, userId)
	req, err := api.NewGetRequest(u)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-Id", api.config.Creds.ClientID)
	req.Header.Set("Authorization", api.GetToken())
	resp, err := api.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var streams Streams
	if err := json.Unmarshal(b, &streams); err != nil {
		return nil, err
	}
	return &streams, nil
}
