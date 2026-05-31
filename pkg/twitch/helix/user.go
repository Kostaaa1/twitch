package helix

import (
	"context"
	"fmt"
	"net/http"
)

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

	// PrimaryColorHex string    `json:"primaryColorHex"`
	// IsPartner       bool      `json:"isPartner"`
	// LastBroadcast   Broadcast `json:"lastBroadcast"`
	// Stream          any       `json:"stream"`
	// Followers       Followers `json:"followers"`
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
