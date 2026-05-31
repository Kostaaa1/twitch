package helix

import (
	"context"
	"fmt"
	"net/http"
)

type Channel struct {
	ID                          string   `json:"id"`
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
