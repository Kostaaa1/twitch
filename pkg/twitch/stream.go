package twitch

import (
	"fmt"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/m3u8"
)

type StreamMetadata struct {
	ID                string `json:"id"`
	BroadcastSettings struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"broadcastSettings"`
	Stream struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"createdAt"`
		Game      struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"game"`
	} `json:"stream"`
}

func (tw *Client) StreamMetadata(channelName string) (*StreamMetadata, error) {
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

	type payload struct {
		Data struct {
			Stream StreamMetadata `json:"user"`
		} `json:"data"`
	}
	var resp payload

	body := strings.NewReader(fmt.Sprintf(gqlPl, channelName))
	if err := tw.sendGqlLoadAndDecode(body, &resp); err != nil {
		return nil, err
	}

	return &resp.Data.Stream, nil
}

func (tw *Client) GetLivestreamCreds(id string) (string, string, error) {
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

	type payload struct {
		Data struct {
			VideoPlaybackAccessToken VideoCredResponse `json:"streamPlaybackAccessToken"`
		} `json:"data"`
	}
	var data payload

	body := strings.NewReader(fmt.Sprintf(gqlPl, id))
	if err := tw.sendGqlLoadAndDecode(body, &data); err != nil {
		return "", "", err
	}

	return data.Data.VideoPlaybackAccessToken.Value, data.Data.VideoPlaybackAccessToken.Signature, nil
}

func (tw *Client) GetStreamMasterPlaylist(channel string) (*m3u8.MasterPlaylist, error) {
	tok, sig, err := tw.GetLivestreamCreds(channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestream credentials: %w", err)
	}

	u := fmt.Sprintf("%s/api/channel/hls/%s.m3u8?token=%s&sig=%s&allow_audio_only=true&allow_source=true", usherURL, channel, tok, sig)

	b, err := tw.fetch(u)
	if err != nil {
		return nil, err
	}

	return m3u8.Master(b), nil
}
