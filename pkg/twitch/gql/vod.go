package gql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
)

func (tw *Client) VideoPlaybackAccessToken(ctx context.Context, id string) (*PlaybackAccessToken, error) {
	gqlPayload := `{
	    "operationName": "PlaybackAccessToken_Template",
	    "query": "query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isLive) {    value    signature   authorization { isForbidden forbiddenReasonCode }   __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isVod) {    value    signature   __typename  }}",
	    "variables": {
	        "isLive": false,
	        "login": "",
	        "isVod": true,
	        "vodID": "%s",
	        "playerType": "site"
	    }
	}`

	var data VideoPlaybackAccessToken_Template
	if err := sendGqlLoadAndDecode(ctx, tw.http, &data, gqlPayload, id); err != nil {
		return nil, err
	}

	at := data.PlaybackAccessToken

	if at.Value == "" || at.Signature == "" {
		return nil, fmt.Errorf("[VOD expired] sorry. Unless you've got a time machine, that content is unavailable")
	}

	return &at, nil
}

func (tw *Client) VideoCommentsByOffsetOrCursor(
	ctx context.Context,
	vodID string,
	offset int,
) (*VideoCommentsByOffsetOrCursor, error) {
	gqlPayload := `{
        "operationName": "VideoCommentsByOffsetOrCursor",
        "variables": {
            "videoID": "%s",
            "contentOffsetSeconds": %d
        },
        "extensions": {
            "persistedQuery": {
                "version": 1,
                "sha256Hash": "b70a3591ff0f4e0313d126c6a1502d79a1c02baebb288227c582044aa76adf6a"
            }
        }
    }`

	var comments VideoCommentsByOffsetOrCursor

	if err := sendGqlLoadAndDecode(
		ctx,
		tw.http,
		&comments,
		gqlPayload,
		vodID,
		offset,
	); err != nil {
		return nil, err
	}

	return &comments, nil
}

func (tw *Client) VideoMetadata(ctx context.Context, vodID string) (*VideoMetadata, error) {
	gqlPayload := `{
		"operationName": "VideoMetadata",
		"variables": {
			"channelLogin": "",
			"videoID": "%s"
		},
		"extensions": {
			"persistedQuery": {
				"version": 1,
				"sha256Hash": "45111672eea2e507f8ba44d101a61862f9c56b11dee09a15634cb75cb9b9084d"
			}
		}
	}`

	var vod VideoMetadata
	if err := sendGqlLoadAndDecode(ctx, tw.http, &vod, gqlPayload, vodID); err != nil {
		return nil, err
	}

	if vod.Video.ID == "" {
		return nil, fmt.Errorf("failed to get the video data for %s", vodID)
	}

	return &vod, nil
}

type FilterableVideoTowerArgs struct {
	Limit           int
	Channel         string
	BroadcasterType helix.VideoBroadcastType
	VideoSort       helix.VideoSort
}

func (args *FilterableVideoTowerArgs) Validate() error {
	if args.Limit < 1 || args.Limit > 100 {
		return fmt.Errorf("invalid limit %d: must be between 1 and 100", args.Limit)
	}

	args.Channel = strings.ToLower(strings.TrimSpace(args.Channel))
	if args.Channel == "" {
		return errors.New("channel login must not be empty")
	}

	switch args.BroadcasterType {
	case "":
		args.BroadcasterType = helix.VideoTypeArchive
	case helix.VideoTypeArchive, helix.VideoTypeHighlight, helix.VideoTypeUpload:
	default:
		return fmt.Errorf("invalid broadcast type %q: must be one of archive, highlight, upload", args.BroadcasterType)
	}

	switch args.VideoSort {
	case "":
		args.VideoSort = helix.SortTime
	case helix.SortTime, helix.SortViews:
	default:
		return fmt.Errorf("invalid video sort %q: must be one of time, views", args.VideoSort)
	}

	return nil
}

// type broadcastType int // ARCHIVE | HIGHLIGHT | CLIP | LIVE
// type videoSort int // TIME | VIEWS
// cursor example: 2732435300|877053|2026-03-26T20:03:29Z|24387
func (tw *Client) FilterableVideoTower_Videos(ctx context.Context, args FilterableVideoTowerArgs) (*FilterableVideoTower_Videos, error) {
	gqlPl := `{
        "operationName": "FilterableVideoTower_Videos",
        "variables": {
            "includePreviewBlur": false,
            "limit": %d,
            "channelOwnerLogin": "%s",
            "broadcastType": "%s",
            "videoSort": "%s"
        },
        "extensions": {
            "persistedQuery": {
                "version": 1,
                "sha256Hash": "67004f7881e65c297936f32c75246470629557a393788fb5a69d6d9a25a8fd5f"
            }
        }
    }`

	var videos FilterableVideoTower_Videos
	if err := sendGqlLoadAndDecode(
		ctx,
		tw.http,
		&videos,
		gqlPl,
		args.Limit,
		args.Channel,
		strings.ToUpper(string(args.BroadcasterType)),
		strings.ToUpper(string(args.VideoSort)),
	); err != nil {
		return nil, err
	}

	return &videos, nil
}

func (gql *Client) SeekPreviewsURL(ctx context.Context, vodID string) (string, string, error) {
	gqlPl := `{
		"query": "query { video(id: \"%s\") { broadcastType, id, createdAt, seekPreviewsURL, owner { login } } }"
	}`
	var data struct {
		Video Video `json:"video"`
	}
	if err := sendGqlLoadAndDecode(ctx, gql.http, &data, gqlPl, vodID); err != nil {
		return "", "", err
	}
	return data.Video.BroadcastType, data.Video.SeekPreviewsURL, nil
}

func (tw *Client) ChannelRoot_AboutPanel(
	ctx context.Context,
	channel string,
) (*ChannelRoot_AboutPanel, error) {
	gqlPl := `{
		"operationName": "ChannelRoot_AboutPanel",
		"variables": {
			"channelLogin": "%s",
			"skipSchedule": true
		},
		"extensions": {
			"persistedQuery": {
				"version": 1,
				"sha256Hash": "3b9cd4edd28e8e6f7ba6152a56157bc2b1c1a8f6e81d70808ad1b85250e5288f"
			}
		}
	}`

	var about ChannelRoot_AboutPanel
	if err := sendGqlLoadAndDecode(ctx, tw.http, &about, gqlPl, channel); err != nil {
		return nil, err
	}

	return &about, nil
}
