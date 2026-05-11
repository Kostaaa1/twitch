package twitch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/twitch/m3u8"
)

func (tw *Client) IsChannelLive(ctx context.Context, channelName string) (bool, error) {
	data, err := tw.StreamMetadata(ctx, channelName)
	if err != nil {
		return false, fmt.Errorf("failed to get the stream metadata for user: %s. error: %v", channelName, err)
	}
	return len(data.ID) > 0, nil
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

	type payload struct {
		Data struct {
			User UseLiveBroadcast `json:"user"`
		} `json:"data"`
	}
	var resp payload

	body := strings.NewReader(fmt.Sprintf(gqlPl, channelName))
	if err := tw.sendGqlLoadAndDecode(ctx, body, &resp); err != nil {
		return nil, err
	}

	return &resp.Data.User, nil
}

func (tw *Client) StreamMetadata(ctx context.Context, channelName string) (*Video, error) {
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

	var resp NielsenContentMetadata

	body := strings.NewReader(fmt.Sprintf(gqlPl, channelName))
	if err := tw.sendGqlLoadAndDecode(ctx, body, &resp); err != nil {
		return nil, err
	}

	return &resp.Data.Video, nil
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

	var data PlaybackAccessToken_Template
	body := strings.NewReader(fmt.Sprintf(gqlPl, channel))

	if err := tw.sendGqlLoadAndDecode(ctx, body, &data); err != nil {
		return nil, err
	}

	return &data.Data.PlaybackAccessToken, nil
}

func (tw *Client) MasterPlaylistStream(ctx context.Context, channel string) (*m3u8.MasterPlaylist, error) {
	tok, err := tw.StreamPlaybackAccessToken(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestream credentials: %w", err)
	}

	url := fmt.Sprintf("%s/api/channel/hls/%s.m3u8?token=%s&sig=%s&allow_audio_only=true&allow_source=true", usherURL, channel, tok.Value, tok.Signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := tw.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return m3u8.Master(b), nil
}
