package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/twitch/m3u8"
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

	body := strings.NewReader(fmt.Sprintf(gqlPayload, id))

	type payload struct {
		Data struct {
			PlaybackAccessToken PlaybackAccessToken `json:"videoPlaybackAccessToken"`
		} `json:"data"`
	}
	var p payload

	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
		return nil, err
	}

	if p.Data.PlaybackAccessToken.Value == "" && p.Data.PlaybackAccessToken.Signature == "" {
		return nil, fmt.Errorf("[VOD expired] sorry. Unless you've got a time machine, that content is unavailable")
	}

	return &p.Data.PlaybackAccessToken, nil
}

func (tw *Client) VideoCommentsByOffsetOrCursor(ctx context.Context, vodID string, offset int) (*VideoCommentsByOffsetOrCursor, error) {
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

	body := strings.NewReader(fmt.Sprintf(gqlPayload, vodID, offset))

	var p VideoCommentsByOffsetOrCursor
	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// func (tw *Client) StreamPlaybackAccessToken(ctx context.Context, login string) (*PlaybackAccessToken, error) {
// 	gqlPayload := `{
// 	    "operationName": "PlaybackAccessToken_Template",
// 	    "query": "query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isLive) {    value    signature   authorization { isForbidden forbiddenReasonCode }   __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isVod) {    value    signature   __typename  }}",
// 	    "variables": {
// 			"isLive": true,
// 			"login": "%s",
// 			"isVod": false,
// 			"vodID": "",
// 			"playerType": "site",
// 			"platform": "web"
// 		}
// 	}`

// 	body := strings.NewReader(fmt.Sprintf(gqlPayload, login))
// 	type payload struct {
// 		Data struct {
// 			PlaybackAccessToken PlaybackAccessToken `json:"videoPlaybackAccessToken"`
// 		} `json:"data"`
// 	}
// 	var p payload

// 	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
// 		return err
// 	}

// 	return nil
// }

func (tw *Client) mockMasterPlaylist(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	subVOD, err := tw.SubVodData(ctx, vodID)
	if err != nil {
		return nil, err
	}
	previewURL, err := url.Parse(subVOD.Video.SeekPreviewsURL)
	if err != nil {
		return nil, err
	}
	return m3u8.MasterPlaylistMock(tw.http, vodID, previewURL, subVOD.Video.BroadcastType), nil
}

func (tw *Client) MasterPlaylistVOD(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	tok, err := tw.VideoPlaybackAccessToken(ctx, vodID)
	if err != nil {
		return nil, err
	}

	m3u8Url := fmt.Sprintf("%s/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true", usherURL, vodID, tok.Value, tok.Signature)

	resp, err := tw.request(ctx, m3u8Url, http.MethodGet, nil, nil)
	if resp.StatusCode == http.StatusForbidden {
		return tw.mockMasterPlaylist(ctx, vodID)
	}

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

func (tw *Client) VideoMetadata(ctx context.Context, id string) (VideoMetadata, error) {
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

	type payload struct {
		Data VideoMetadata `json:"data"`
	}
	var p payload

	body := strings.NewReader(fmt.Sprintf(gqlPayload, id))

	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
		return VideoMetadata{}, err
	}

	if p.Data.Video.ID == "" {
		return VideoMetadata{}, fmt.Errorf("failed to get the video data for %s", id)
	}

	return p.Data, nil
}

// REVISIT: ..
func (tw *Client) ListVideosByChannelName(ctx context.Context, channel string, limit int) ([]FilterableVideoTower_Videos, error) {
	if limit > 100 {
		return nil, errors.New("limit value must be between 1 and 100")
	}

	gqlPl := `{
		"operationName": "FilterableVideoTower_Videos",
		"variables": {
			"limit": %d,
			"channelOwnerLogin": "%s",
			"broadcastType": "ARCHIVE",
			"videoSort": "TIME"
		},
		"extensions": {
			"persistedQuery": {
				"version": 1,
				"sha256Hash": "acea7539a293dfd30f0b0b81a263134bb5d9a7175592e14ac3f7c77b192de416"
			}
		}
	}`

	body := strings.NewReader(fmt.Sprintf(gqlPl, limit, channel))

	var videos []FilterableVideoTower_Videos
	if err := tw.sendGqlLoadAndDecode(ctx, body, &videos); err != nil {
		return nil, err
	}

	return videos, nil
}

func (tw *Client) SubVodData(ctx context.Context, vodID string) (SubVODResponse, error) {
	gqlPayload := `{
 	   "query": "query { video(id: \"%s\") { broadcastType, createdAt, seekPreviewsURL, owner { login } } }"
	}`
	body := strings.NewReader(fmt.Sprintf(gqlPayload, vodID))

	var subVodResponse struct {
		Data SubVODResponse `json:"data"`
	}

	if err := tw.sendGqlLoadAndDecode(ctx, body, &subVodResponse); err != nil {
		return SubVODResponse{}, err
	}

	return subVodResponse.Data, nil
}

func (tw *Client) GQLTest(ctx context.Context) {
	gqlPayload := `{
 	   "query": "query { video(id: \"%s\") { broadcastType, createdAt, seekPreviewsURL, owner { login } } }"
	}`

	body := strings.NewReader(fmt.Sprintf(gqlPayload, "2766330803"))
	var dst interface{}

	h := http.Header{}
	h.Set("Client-Id", "kimne78kx3ncx6brgo4mv6wki5h1ko")
	h.Set("Content-Type", "application/json")

	if err := tw.FetchWithDecode(context.Background(), "https://gql.twitch.tv/gql", http.MethodPost, body, &dst, h); err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(dst, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

}
