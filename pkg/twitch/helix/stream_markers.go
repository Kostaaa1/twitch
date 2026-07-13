package helix

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Reference: https://dev.twitch.tv/docs/api/reference#twitch-api-reference
// Requires a user access token that includes the user:read:broadcast or channel:manage:broadcast scope.

// Required: user_id, video_id
// Opt: first, before, after

type Marker struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	Description     string    `json:"description"`
	PositionSeconds int       `json:"position_seconds"`
	URL             string    `json:"URL"`
}

type MarkerData struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserLogin string `json:"user_login"`
	Videos    []struct {
		VideoID string   `json:"video_id"`
		Markers []Marker `json:"markers"`
	} `json:"videos"`
}

type StreamMarkers struct {
	Data       []MarkerData `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

type streamMarkers struct {
	c      *Client
	url    *url.URL
	values url.Values
}

// func (f *streamMarkers) UserID(userID string) *streamMarkers {
// 	f.values.Add("user_id", userID)
// 	return f
// }
// func (f *streamMarkers) VideoID(videoID string) *streamMarkers {
// 	f.values.Add("video_id", videoID)
// 	return f
// }

func (f *streamMarkers) First(first int) *streamMarkers {
	f.values.Add("first", strconv.Itoa(first))
	return f
}
func (f *streamMarkers) Before(cursor string) *streamMarkers {
	f.values.Add("before", cursor)
	return f
}
func (f *streamMarkers) After(cursor string) *streamMarkers {
	f.values.Add("after", cursor)
	return f
}

func (s *streamMarkers) Run(ctx context.Context) ([]StreamMarkers, error) {
	s.url.RawQuery = s.values.Encode()
	var body helixPaginatedEnvelope[StreamMarkers]
	err := s.c.RequestWithAccessToken(ctx, s.url.String(), http.MethodGet, nil, &body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *Client) StreamMarkers(userID, videoID string) *streamMarkers {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/streams/markers")
	v := url.Values{}
	v.Set("user_id", userID)
	v.Set("video_id", videoID)
	return &streamMarkers{
		c:      c,
		url:    parsed,
		values: v,
	}
}
