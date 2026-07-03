package helix

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type streamType string

const (
	LiveType streamType = "live"
	AllType  streamType = "all"
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

type stream struct {
	c      *Client
	url    *url.URL
	values url.Values
}

func (s *stream) UserID(id string) *stream {
	s.values.Add("user_id", id)
	return s
}
func (s *stream) UserLogin(loginName string) *stream {
	s.values.Add("user_login", loginName)
	return s
}
func (s *stream) GameID(gameID string) *stream {
	s.values.Add("game_id", gameID)
	return s
}
func (s *stream) Type(t streamType) *stream {
	s.values.Add("type", string(t))
	return s
}
func (s *stream) Language(language string) *stream {
	s.values.Add("language", language)
	return s
}
func (f *stream) First(first int) *stream {
	f.values.Add("first", strconv.Itoa(first))
	return f
}
func (f *stream) Before(cursor string) *stream {
	f.values.Add("before", cursor)
	return f
}
func (f *stream) After(cursor string) *stream {
	f.values.Add("after", cursor)
	return f
}

func (s *stream) Run(ctx context.Context) (*helixPaginatedEnvelope[Stream], error) {
	s.url.RawQuery = s.values.Encode()
	var body helixPaginatedEnvelope[Stream]
	if err := s.c.Request(ctx, s.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Client) Stream() *stream {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/streams")
	return &stream{
		c:      c,
		url:    parsed,
		values: url.Values{},
	}
}
