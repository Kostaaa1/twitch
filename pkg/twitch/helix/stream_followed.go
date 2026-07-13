package helix

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Get Followed Streams
// Gets the list of broadcasters that the user follows and that are streaming live.

// Authorization
// Requires a user access token that includes the user:read:follows scope.

// URL
// GET https://api.twitch.tv/helix/streams/followed

type followedStreams struct {
	c      *Client
	url    *url.URL
	values url.Values
}

func (c *followedStreams) First(first int) *followedStreams {
	c.values.Add("first", strconv.Itoa(first))
	return c
}

func (c *followedStreams) After(cursor string) *followedStreams {
	c.values.Add("after", cursor)
	return c
}

func (s *followedStreams) Run(ctx context.Context) (*helixPaginatedEnvelope[Stream], error) {
	s.url.RawQuery = s.values.Encode()
	var body helixPaginatedEnvelope[Stream]
	err := s.c.RequestWithAccessToken(ctx, s.url.String(), http.MethodGet, nil, &body)
	if err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Client) FollowedStreams(userID string) *followedStreams {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/streams/followed")
	v := url.Values{}
	v.Add("user_id", userID)
	return &followedStreams{
		c:      c,
		url:    parsed,
		values: v,
	}
}
