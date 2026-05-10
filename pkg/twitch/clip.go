package twitch

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

func (client *Client) ConstructUsherURL(clip PlaybackAccessToken, sourceURL string) (string, error) {
	return fmt.Sprintf("%s?sig=%s&token=%s", sourceURL, url.QueryEscape(clip.Signature), url.QueryEscape(clip.Value)), nil
}

func (client *Client) ClipMetadata(ctx context.Context, slug string) (*Clip, error) {
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

	var result struct {
		Data struct {
			Clip Clip `json:"clip"`
		} `json:"data"`
	}

	body := strings.NewReader(fmt.Sprintf(gqlPayload, slug))
	if err := client.sendGqlLoadAndDecode(ctx, body, &result); err != nil {
		return nil, err
	}

	if result.Data.Clip.ID == "" {
		return nil, fmt.Errorf("failed to get the clip data for %s", slug)
	}

	return &result.Data.Clip, nil
}
