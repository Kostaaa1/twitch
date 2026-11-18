package gql

import (
	"context"
	"fmt"
)

func (gql *Client) ShelfContentEdge() {
}

// func (tw *Client) IsChannelLive(ctx context.Context, channelName string) (bool, error) {
// 	data, err := tw.StreamMetadata(ctx, channelName)
// 	if err != nil {
// 		return false, fmt.Errorf("failed to get the stream metadata for user: %s. error: %v", channelName, err)
// 	}
// 	return len(data.User.Stream.ID) > 0, nil
// }

func (tw *Client) UseLiveBroadcast(ctx context.Context, channelName string) (*UseLiveBroadcast, error) {
	payload := Payload{
		OperationName: "UseLiveBroadcast",
		Variables: map[string]any{
			"channelLogin": channelName,
		},
		Extensions: commonExtensions("0b47cc6d8c182acd2e78b81c8ba5414a5a38057f2089b1bbcfa6046aae248bd2"),
	}

	var broadcast UseLiveBroadcast
	if err := sendGqlLoadAndDecode(ctx, tw.http, &broadcast, payload); err != nil {
		return nil, err
	}

	return &broadcast, nil
}

func (tw *Client) StreamMetadata(ctx context.Context, channel string) (*NielsenContentMetadata, error) {
	payload := Payload{
		OperationName: "NielsenContentMetadata",
		Variables: map[string]any{
			"isCollectionContent": false,
			"isLiveContent":       true,
			"isVODContent":        false,
			"collectionID":        "",
			"login":               channel,
			"vodID":               "",
		},
		Extensions: commonExtensions("2dbf505ee929438369e68e72319d1106bb3c142e295332fac157c90638968586"),
	}

	var stream NielsenContentMetadata
	if err := sendGqlLoadAndDecode(ctx, tw.http, &stream, payload); err != nil {
		return nil, err
	}

	return &stream, nil
}

func (tw *Client) StreamPlaybackAccessToken(ctx context.Context, channel string) (*PlaybackAccessToken, error) {
	payload := Payload{
		OperationName: "PlaybackAccessToken_Template",
		Query:         playbackAccessTokenQuery,
		Variables: map[string]any{
			"isLive":     true,
			"login":      channel,
			"isVod":      false,
			"vodID":      "",
			"playerType": "site",
		},
	}

	var data StreamPlaybackAccessToken_Template
	if err := sendGqlLoadAndDecode(ctx, tw.http, &data, payload); err != nil {
		return nil, err
	}

	at := data.PlaybackAccessToken

	if at.Value == "" || at.Signature == "" {
		return nil, fmt.Errorf("[STREAM expired] sorry. Unless you've got a time machine, that content is unavailable")
	}

	return &at, nil
}
