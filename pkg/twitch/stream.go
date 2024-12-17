package twitch

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/m3u8"
	"github.com/Kostaaa1/twitch/internal/spinner"
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
	isLive, err := api.IsChannelLive(channel)
	if err != nil {
		return nil, err
	}
	if !isLive {
		return nil, fmt.Errorf("%s is offline", channel)
	}

	tok, sig, err := api.GetLivestreamCreds(channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestream credentials: %w", err)
	}

	u := fmt.Sprintf("%s/api/channel/hls/%s.m3u8?token=%s&sig=%s&allow_audio_only=true&allow_source=true", usherURL, channel, tok, sig)

	resp, err := api.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if s := resp.StatusCode; s < 200 || s >= 300 {
		return nil, fmt.Errorf("unsupported status code (%v) for url: %s", s, u)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return m3u8.Master(b), nil
}

func (api *API) RecordStream(unit MediaUnit) error {
	isLive, err := api.IsChannelLive(unit.Slug)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", unit.Slug)
	}

	master, err := api.GetStreamMasterPlaylist(unit.Slug)
	if err != nil {
		return err
	}

	mediaList, err := master.GetVariantPlaylistByQuality(unit.Quality)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	tickCount := 0
	var halfBytes *bytes.Reader

	for {
		select {
		case <-ticker.C:
			tickCount++
			var n int64

			if tickCount%2 != 0 {
				b, err := api.fetch(mediaList.URL)
				if err != nil {
					return fmt.Errorf("failed to fetch playlist: %w", err)
				}

				segments := strings.Split(string(b), "\n")
				tsURL := segments[len(segments)-2]

				bodyBytes, err := api.fetch(tsURL)
				if err != nil {
					return err
				}

				half := len(bodyBytes) / 2
				halfBytes = bytes.NewReader(bodyBytes[half:])

				n, err = io.Copy(unit.W, bytes.NewReader(bodyBytes[:half]))
				if err != nil {
					return err
				}
			}

			if tickCount%2 == 0 && halfBytes.Len() > 0 {
				n, err = io.Copy(unit.W, halfBytes)
				if err != nil {
					return err
				}
				halfBytes.Reset([]byte{})
			}

			if f, ok := unit.W.(*os.File); ok && f != nil {
				api.progressCh <- spinner.ChannelMessage{
					Text:  f.Name(),
					Bytes: n,
				}
			}
		}
	}
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
