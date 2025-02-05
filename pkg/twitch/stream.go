package twitch

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Kostaaa1/twitch/internal/m3u8"
)

func (api *API) GetLivestreamCreds(id string) (string, string, error) {
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
	if err := api.sendGqlLoadAndDecode(body, &data); err != nil {
		return "", "", err
	}
	return data.Data.VideoPlaybackAccessToken.Value, data.Data.VideoPlaybackAccessToken.Signature, nil
}

func (api *API) GetStreamMasterPlaylist(channel string) (*m3u8.MasterPlaylist, error) {
	tok, sig, err := api.GetLivestreamCreds(channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestream credentials: %w", err)
	}

	u := fmt.Sprintf("%s/api/channel/hls/%s.m3u8?token=%s&sig=%s&allow_audio_only=true&allow_source=true", usherURL, channel, tok, sig)

	b, err := api.fetch(u)
	if err != nil {
		return nil, err
	}

	return m3u8.Master(b), nil
}

func (api *API) OpenStreamInMediaPlayer(channel string) error {
	master, err := api.GetStreamMasterPlaylist(channel)
	if err != nil {
		return err
	}
	list, err := master.GetVariantPlaylistByQuality("best")
	if err != nil {
		return err
	}
	cmd := exec.Command("vlc", list.URL)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
