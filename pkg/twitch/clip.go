package twitch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (tw *Client) ConstructUsherURL(clip PlaybackAccessToken, sourceURL string) (string, error) {
	return fmt.Sprintf("%s?sig=%s&token=%s", sourceURL, url.QueryEscape(clip.Signature), url.QueryEscape(clip.Value)), nil
}

// channel clips
// filter ALL_TIME, LAST_WEEK, LAST_DAY, LAST_MONTH
// cursor base64 offset - 20, 40 per limit
func (tw *Client) ClipsCardsUser(
	ctx context.Context,
	channel string,
	limit int,
	filter string,
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
		tw.http,
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

func (tw *Client) ClipMetadata(ctx context.Context, slug string) (*Clip, error) {
	gqlPayload := `{
        "operationName": "ShareClipRenderStatus",
        "variables": {
            "slug": "%s"
        },
        "extensions": {
            "persistedQuery": {
                "version": 1,
                "sha256Hash": "f130048a462a0ac86bb54d653c968c514e9ab9ca94db52368c1179e97b0f16eb"
            }
        }
    }`

	var clip Clip
	if err := sendGqlLoadAndDecode(ctx, tw.http, &clip, gqlPayload, slug); err != nil {
		return nil, err
	}

	if clip.ID == "" {
		return nil, fmt.Errorf("failed to get the clip data for %s", slug)
	}

	return &clip, nil
}
