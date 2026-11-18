package gql

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type VideoSort string
type VideoBroadcastType string

const (
	SortTime           VideoSort          = "TIME"
	SortViews          VideoSort          = "VIEWS"
	SortTrending       VideoSort          = "TRENDING"
	VideoTypeAll       VideoBroadcastType = "ALL"       // Past broadcasts
	VideoTypeArchive   VideoBroadcastType = "ARCHIVE"   // Past broadcasts
	VideoTypeHighlight VideoBroadcastType = "HIGHLIGHT" // Highlights
	VideoTypeUpload    VideoBroadcastType = "UPLOAD"    // Uploads
)

// playbackAccessTokenQuery is shared by stream and video access token requests;
// isLive/isVod variables select which token is returned.
const playbackAccessTokenQuery = `query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {
	streamPlaybackAccessToken(channelName: $login, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isLive) {
		value
		signature
		authorization { isForbidden forbiddenReasonCode }
		__typename
	}
	videoPlaybackAccessToken(id: $vodID, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isVod) {
		value
		signature
		__typename
	}
}`

func (tw *Client) VideoPlaybackAccessToken(ctx context.Context, id string) (*PlaybackAccessToken, error) {
	payload := Payload{
		OperationName: "PlaybackAccessToken_Template",
		Query:         playbackAccessTokenQuery,
		Variables: map[string]any{
			"isLive":     false,
			"login":      "",
			"isVod":      true,
			"vodID":      id,
			"playerType": "site",
		},
	}

	var data VideoPlaybackAccessToken_Template
	if err := sendGqlLoadAndDecode(ctx, tw.http, &data, payload); err != nil {
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
	payload := Payload{
		OperationName: "VideoCommentsByOffsetOrCursor",
		Variables: map[string]any{
			"videoID":              vodID,
			"contentOffsetSeconds": offset,
		},
		Extensions: commonExtensions("b70a3591ff0f4e0313d126c6a1502d79a1c02baebb288227c582044aa76adf6a"),
	}

	var comments VideoCommentsByOffsetOrCursor

	if err := sendGqlLoadAndDecode(ctx, tw.http, &comments, payload); err != nil {
		return nil, err
	}

	return &comments, nil
}

func (tw *Client) VideoMetadata(ctx context.Context, vodID string) (*VideoMetadata, error) {
	payload := Payload{
		OperationName: "VideoMetadata",
		Variables: map[string]any{
			"channelLogin": "",
			"videoID":      vodID,
		},
		Extensions: commonExtensions("45111672eea2e507f8ba44d101a61862f9c56b11dee09a15634cb75cb9b9084d"),
	}

	var vod VideoMetadata
	if err := sendGqlLoadAndDecode(ctx, tw.http, &vod, payload); err != nil {
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
	BroadcasterType VideoBroadcastType
	VideoSort       VideoSort
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
		args.BroadcasterType = VideoTypeArchive
	case VideoTypeArchive, VideoTypeHighlight, VideoTypeUpload:
	default:
		return fmt.Errorf("invalid broadcast type %q: must be one of archive, highlight, upload", args.BroadcasterType)
	}

	switch args.VideoSort {
	case "":
		args.VideoSort = SortTime
	case SortTime, SortViews:
	default:
		return fmt.Errorf("invalid video sort %q: must be one of time, views", args.VideoSort)
	}

	return nil
}

// type broadcastType int // ARCHIVE | HIGHLIGHT | CLIP | LIVE
// type videoSort int // TIME | VIEWS
// cursor example: 2732435300|877053|2026-03-26T20:03:29Z|24387
func (tw *Client) FilterableVideoTower_Videos(ctx context.Context, args FilterableVideoTowerArgs) (*FilterableVideoTower_Videos, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	payload := Payload{
		OperationName: "FilterableVideoTower_Videos",
		Variables: map[string]any{
			"includePreviewBlur": false,
			"limit":              args.Limit,
			"channelOwnerLogin":  args.Channel,
			"broadcastType":      args.BroadcasterType,
			"videoSort":          args.VideoSort,
		},
		Extensions: commonExtensions("67004f7881e65c297936f32c75246470629557a393788fb5a69d6d9a25a8fd5f"),
	}

	var videos FilterableVideoTower_Videos
	if err := sendGqlLoadAndDecode(ctx, tw.http, &videos, payload); err != nil {
		return nil, err
	}

	return &videos, nil
}

func (gql *Client) SeekPreviewsURL(ctx context.Context, vodID string) (string, string, error) {
	payload := Payload{
		Query:     `query($id: ID) { video(id: $id) { broadcastType, id, createdAt, seekPreviewsURL, owner { login } } }`,
		Variables: map[string]any{"id": vodID},
	}

	var data struct {
		Video Video `json:"video"`
	}
	if err := sendGqlLoadAndDecode(ctx, gql.http, &data, payload); err != nil {
		return "", "", err
	}
	return data.Video.BroadcastType, data.Video.SeekPreviewsURL, nil
}

func (tw *Client) ChannelRoot_AboutPanel(
	ctx context.Context,
	channel string,
) (*ChannelRoot_AboutPanel, error) {
	payload := Payload{
		OperationName: "ChannelRoot_AboutPanel",
		Variables: map[string]any{
			"channelLogin": channel,
			"skipSchedule": true,
		},
		Extensions: commonExtensions("3b9cd4edd28e8e6f7ba6152a56157bc2b1c1a8f6e81d70808ad1b85250e5288f"),
	}

	var about ChannelRoot_AboutPanel
	if err := sendGqlLoadAndDecode(ctx, tw.http, &about, payload); err != nil {
		return nil, err
	}

	return &about, nil
}
