package gql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type criteriaFilter string

const (
	AllTime   criteriaFilter = "ALL_TIME"
	LastDay   criteriaFilter = "LAST_DAY"
	LastWeek  criteriaFilter = "LAST_WEEK"
	LastMonth criteriaFilter = "LAST_MONTH"
	LastYear  criteriaFilter = "LAST_YEAR"
)

func (c *Client) ConstructUsherURL(clip PlaybackAccessToken, sourceURL string) (string, error) {
	return fmt.Sprintf("%s?sig=%s&token=%s", sourceURL, url.QueryEscape(clip.Signature), url.QueryEscape(clip.Value)), nil
}

func (c *Client) ClipsCardsUser(
	ctx context.Context,
	channel string,
	limit int,
	filter criteriaFilter,
) (*ClipsCardsUser, error) {
	if limit > 100 {
		return nil, errors.New("limit value must be between 1 and 100")
	}

	gqlPl := `{
		"operationName": "ClipsCards__User",
		"variables": {
			"login": "%s",
			"limit": %d,
			"criteria": {
				"filter": "%s",
				"shouldFilterByDiscoverySetting": true
			},
			"cursor": null
		},
		"extensions": {
			"persistedQuery": {
				"version": 1,
				"sha256Hash": "1cd671bfa12cec480499c087319f26d21925e9695d1f80225aae6a4354f23088"
			}
		}
	}`

	var card ClipsCardsUser
	if err := sendGqlLoadAndDecode(
		ctx,
		c.http,
		&card,
		gqlPl,
		channel,
		limit,
		filter,
	); err != nil {
		return nil, err
	}

	return &card, nil
}

func (c *Client) ClipMetadata(ctx context.Context, slug string) (*Clip, error) {
	gqlPayload := `{
        "operationName": "ShareClipRenderStatus",
        "variables": {
            "slug": "%s"
        },
        "extensions": {
            "persistedQuery": {
                "version": 1,
                "sha256Hash": "324783ea014524fa10a88739aa507de7a52f9624574dba9739a52b8c97d885cf"
            }
        }
    }`

	var data struct {
		Clip Clip `json:"clip"`
	}

	if err := sendGqlLoadAndDecode(ctx, c.http, &data, gqlPayload, slug); err != nil {
		return nil, err
	}

	if data.Clip.ID == "" {
		return nil, fmt.Errorf("failed to get the clip data for %s", slug)
	}

	return &data.Clip, nil
}

func (c *Client) ClipTitle(ctx context.Context, slug string) (string, error) {
	gqlPl := `{
		"query": "query { clip(slug: \"%s\") { title } }"
	}`

	var data struct {
		Clip struct {
			Title string `json:"title"`
		} `json:"clip"`
	}
	if err := sendGqlLoadAndDecode(ctx, c.http, &data, gqlPl, slug); err != nil {
		return "", err
	}

	return data.Clip.Title, nil
}

func (c *Client) VideoTitle(ctx context.Context, vodID string) (string, error) {
	gqlPl := `{
		"query": "query { video(id: \"%s\") { title } }"
	}`

	var data struct {
		Video struct {
			Title string `json:"title"`
		} `json:"video"`
	}

	if err := sendGqlLoadAndDecode(ctx, c.http, &data, gqlPl, vodID); err != nil {
		return "", err
	}

	return data.Video.Title, nil
}

func (c *Client) StreamTitle(ctx context.Context, channel string) (string, error) {
	gqlPl := `{
		"query": "query { user(login: \"%s\") { stream { __typename } } }"
	}`

	var data interface{}
	if err := sendGqlLoadAndDecode(ctx, c.http, &data, gqlPl, channel); err != nil {
		return "", err
	}

	b, _ := json.MarshalIndent(data, "", " ")
	fmt.Println(string(b))

	return "dskao", nil
}
