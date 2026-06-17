package helix

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Get Followed Streams
// Gets the list of broadcasters that the user follows and that are streaming live.

// Authorization
// Requires a user access token that includes the user:read:follows scope.

// URL
// GET https://api.twitch.tv/helix/streams/followed

type Video struct {
	ID            string      `json:"id"`
	StreamID      interface{} `json:"stream_id"`
	UserID        string      `json:"user_id"`
	UserLogin     string      `json:"user_login"`
	UserName      string      `json:"user_name"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	CreatedAt     time.Time   `json:"created_at"`
	PublishedAt   time.Time   `json:"published_at"`
	URL           string      `json:"url"`
	ThumbnailURL  string      `json:"thumbnail_url"`
	Viewable      string      `json:"viewable"`
	ViewCount     int         `json:"view_count"`
	Language      string      `json:"language"`
	Type          string      `json:"type"`
	Duration      string      `json:"duration"`
	MutedSegments []struct {
		Duration int `json:"duration"`
		Offset   int `json:"offset"`
	} `json:"muted_segments"`
}

type videoPeriod string
type videoSort string
type videoType string

const (
	PeriodAll   videoPeriod = "all"
	PeriodDay   videoPeriod = "day"
	PeriodMonth videoPeriod = "month"
	PeriodYear  videoPeriod = "year"

	SortTime     videoSort = "time"
	SortTrending videoSort = "trending"
	SortViews    videoSort = "views"

	VideoTypeAll       videoType = "all"
	VideoTypeArchive   videoType = "archive"
	VideoTypeHighlight videoType = "highlight"
	VideoTypeUpload    videoType = "upload"
)

type video struct {
	c      *Client
	url    *url.URL
	values url.Values
}

func (s *video) Language(lang string) *video {
	s.values.Add("language", lang)
	return s
}
func (s *video) Period(p videoPeriod) *video {
	s.values.Add("period", string(p))
	return s
}
func (s *video) Sort(p videoSort) *video {
	s.values.Add("sort", string(p))
	return s
}
func (s *video) Type(p videoType) *video {
	s.values.Add("type", string(p))
	return s
}
func (f *video) First(first int) *video {
	f.values.Add("first", strconv.Itoa(first))
	return f
}
func (f *video) Before(cursor string) *video {
	f.values.Add("before", cursor)
	return f
}
func (f *video) After(cursor string) *video {
	f.values.Add("after", cursor)
	return f
}

func (s *video) Run(ctx context.Context) (*helixPaginatedEnvelope[Video], error) {
	s.url.RawQuery = s.values.Encode()
	var body helixPaginatedEnvelope[Video]
	if err := s.c.Request(ctx, s.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Client) VideosByID(id string) *video {
	return c.videos("id", id)
}

func (c *Client) VideosByGameID(gameID string) *video {
	return c.videos("game_id", gameID)
}

func (c *Client) VideosByUserID(userID string) *video {
	return c.videos("user_id", userID)
}

func (c *Client) videos(k, v string) *video {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/videos")
	params := url.Values{}
	params.Add(k, v)
	return &video{
		c:      c,
		url:    parsed,
		values: params,
	}
}
