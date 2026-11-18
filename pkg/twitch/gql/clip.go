package gql

import (
	"context"
	"errors"
	"fmt"
)

type CriteriaFilter string

const (
	AllTime   CriteriaFilter = "ALL_TIME"
	LastDay   CriteriaFilter = "LAST_DAY"
	LastWeek  CriteriaFilter = "LAST_WEEK"
	LastMonth CriteriaFilter = "LAST_MONTH"
)

type ClipsCardsUserArgs struct {
	Channel string
	Limit   int
	Filter  CriteriaFilter
}

func (a *ClipsCardsUserArgs) Validate() error {
	if a.Limit > 100 {
		return errors.New("limit value must be between 1 and 100")
	}
	return nil
}

func (c *Client) ClipsCardsUser(ctx context.Context, args ClipsCardsUserArgs) (*ClipsCardsUser, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	payload := Payload{
		OperationName: "ClipsCards__User",
		Variables: map[string]any{
			"login": args.Channel,
			"limit": args.Limit,
			"criteria": map[string]any{
				"filter":                         args.Filter,
				"shouldFilterByDiscoverySetting": true,
			},
			"cursor": nil,
		},
		Extensions: commonExtensions("1cd671bfa12cec480499c087319f26d21925e9695d1f80225aae6a4354f23088"),
	}

	var card ClipsCardsUser
	if err := sendGqlLoadAndDecode(ctx, c.http, &card, payload); err != nil {
		return nil, err
	}

	return &card, nil
}

func (c *Client) ClipMetadata(ctx context.Context, slug string) (*Clip, error) {
	payload := Payload{
		OperationName: "ShareClipRenderStatus",
		Variables: map[string]any{
			"slug": slug,
		},
		Extensions: commonExtensions("324783ea014524fa10a88739aa507de7a52f9624574dba9739a52b8c97d885cf"),
	}

	var data struct {
		Clip Clip `json:"clip"`
	}
	if err := sendGqlLoadAndDecode(ctx, c.http, &data, payload); err != nil {
		return nil, err
	}
	if data.Clip.ID == "" {
		return nil, fmt.Errorf("failed to get the clip data for %s", slug)
	}

	return &data.Clip, nil
}

func (c *Client) ClipTitle(ctx context.Context, slug string) (string, error) {
	payload := Payload{
		Query:     `query($slug: ID!) { clip(slug: $slug) { title } }`,
		Variables: map[string]any{"slug": slug},
	}

	var data struct {
		Clip struct {
			Title string `json:"title"`
		} `json:"clip"`
	}
	if err := sendGqlLoadAndDecode(ctx, c.http, &data, payload); err != nil {
		return "", err
	}

	return data.Clip.Title, nil
}

func (c *Client) VideoTitle(ctx context.Context, vodID string) (string, error) {
	payload := Payload{
		Query:     `query($id: ID) { video(id: $id) { title } }`,
		Variables: map[string]any{"id": vodID},
	}

	var data struct {
		Video struct {
			Title string `json:"title"`
		} `json:"video"`
	}

	if err := sendGqlLoadAndDecode(ctx, c.http, &data, payload); err != nil {
		return "", err
	}

	return data.Video.Title, nil
}

func (c *Client) StreamTitle(ctx context.Context, channel string) (string, error) {
	data, err := c.StreamMetadata(ctx, channel)
	if err != nil {
		return "", err
	}
	return data.User.BroadcastSettings.Title, nil
}
