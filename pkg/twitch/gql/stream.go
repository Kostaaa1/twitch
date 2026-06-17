package gql

import (
	"context"
	"fmt"
)

func (tw *Client) IsChannelLive(ctx context.Context, channelName string) (bool, error) {
	data, err := tw.StreamMetadata(ctx, channelName)
	if err != nil {
		return false, fmt.Errorf("failed to get the stream metadata for user: %s. error: %v", channelName, err)
	}
	return len(data.User.ID) > 0, nil
}

func (tw *Client) UseLiveBroadcast(ctx context.Context, channelName string) (*UseLiveBroadcast, error) {
	gqlPl := `{
		"operationName": "UseLiveBroadcast",
		"variables": {
			"channelLogin": "%s"
		},
		"extensions": {
			"persistedQuery": {
			"version": 1,
			"sha256Hash": "0b47cc6d8c182acd2e78b81c8ba5414a5a38057f2089b1bbcfa6046aae248bd2"
			}
		}
	}`

	var broadcast UseLiveBroadcast
	if err := sendGqlLoadAndDecode(ctx, tw.http, &broadcast, gqlPl, channelName); err != nil {
		return nil, err
	}

	return &broadcast, nil
}

func (tw *Client) StreamMetadata(ctx context.Context, channel string) (*NielsenContentMetadata, error) {
	gqlPl := `{
		"operationName": "NielsenContentMetadata",
		"variables": {
			"isCollectionContent": false,
			"isLiveContent": true,
			"isVODContent": false,
			"collectionID": "",
			"login": "%s",
			"vodID": ""
		},
		"extensions": {
			"persistedQuery": {
				"version": 1,
				"sha256Hash": "2dbf505ee929438369e68e72319d1106bb3c142e295332fac157c90638968586"
			}
		}
	}`

	var stream NielsenContentMetadata
	if err := sendGqlLoadAndDecode(ctx, tw.http, &stream, gqlPl, channel); err != nil {
		return nil, err
	}

	return &stream, nil
}

func (tw *Client) StreamPlaybackAccessToken(ctx context.Context, channel string) (*PlaybackAccessToken, error) {
	gqlPl := `{
		"operationName": "PlaybackAccessToken_Template",
		"query": "query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isLive) {    value    signature   authorization { isForbidden forbiddenReasonCode }   __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isVod) {    value    signature   __typename  }}",
		"variables": {
			"isLive": true,
			"login": "%s",
			"isVod": false,
			"vodID": "",
			"playerType": "site"
		}
	}`

	var data StreamPlaybackAccessToken_Template
	if err := sendGqlLoadAndDecode(ctx, tw.http, &data, gqlPl, channel); err != nil {
		return nil, err
	}

	at := data.PlaybackAccessToken

	if at.Value == "" || at.Signature == "" {
		return nil, fmt.Errorf("[STREAM expired] sorry. Unless you've got a time machine, that content is unavailable")
	}

	return &at, nil
}
