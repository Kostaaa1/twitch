package helix

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// List of all streams
// Requires

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

type StreamsCaller struct {
	client *Client
	url    *url.URL
	values url.Values
}

func (s *StreamsCaller) UserID(id string) *StreamsCaller {
	s.values.Add("user_id", id)
	return s
}

func (s *StreamsCaller) UserLogin(loginName string) *StreamsCaller {
	s.values.Add("user_login", loginName)
	return s
}

func (s *StreamsCaller) GameID(gameID string) *StreamsCaller {
	s.values.Add("game_id", gameID)
	return s
}

// all | live
func (s *StreamsCaller) Type(streamType string) *StreamsCaller {
	s.values.Add("type", streamType)
	return s
}

func (s *StreamsCaller) Language(language string) *StreamsCaller {
	s.values.Add("language", language)
	return s
}

// default 20 - max 100
func (s *StreamsCaller) First(num int) *StreamsCaller {
	if num > 100 || num < 0 {
		return nil
	}
	s.values.Add("first", strconv.Itoa(num))
	return s
}

// func (s *StreamsCaller) After(streamType string) {
// }

// func (s *StreamsCaller) Before(streamType string) {
// }

func (s *StreamsCaller) Run(ctx context.Context) ([]Stream, error) {
	s.url.RawQuery = s.values.Encode()
	var body helixEnvelope[Stream]
	if err := s.client.Request(ctx, s.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return body.Data, nil
}

func (c *Client) Streams() *StreamsCaller {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/streams")
	return &StreamsCaller{
		client: c,
		url:    parsed,
		values: url.Values{},
	}
}
