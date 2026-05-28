package gql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kostaaa1/twitch/internal/downloader/m3u8"
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

	var at PlaybackAccessToken
	if err := sendGqlLoadAndDecode(ctx, tw.http, &at, gqlPayload, id); err != nil {
		return nil, err
	}

	if at.Value == "" && at.Signature == "" {
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
		ctx, tw.http,
		&comments,
		gqlPayload,
		vodID,
		offset,
	); err != nil {
		return nil, err
	}

	return &comments, nil
}

func (tw *Client) mockMasterPlaylist(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	bt, previewURL, err := tw.querySeekPreviewsURL(ctx, vodID)
	if err != nil {
		return nil, err
	}

	if previewURL == "" {
		return nil, fmt.Errorf("failed to acquire previewURL for video: %s", vodID)
	}

	u, err := url.Parse(previewURL)
	if err != nil {
		return nil, err
	}

	subdomain := strings.Split(u.Host, ".")[0]
	fmt.Println("Subdomain", subdomain)
	fmt.Println("Preview", previewURL)

	master := m3u8.MasterPlaylist{
		Origin: "s3",
		B:      false,
		Region: "EU",
		UserIP: "127.0.0.1",
		// ServingID:       createServingID(),
		Cluster:         "cloudfront_vod",
		UserCountry:     "BE",
		ManifestCluster: "cloudfront_vod",
	}

	resolutions := map[string]struct {
		Res string
		FPS string
	}{
		"chunked":    {Res: "1920x1080", FPS: "60"},
		"720p60":     {Res: "1280x720", FPS: "60"},
		"720p30":     {Res: "1280x720", FPS: "30"},
		"480p30":     {Res: "854x480", FPS: "30"},
		"360p30":     {Res: "640x360", FPS: "30"},
		"160p30":     {Res: "284x160", FPS: "30"},
		"audio_only": {Res: "audio_only", FPS: ""},
	}

	isQualityValid := func(u string) bool {
		resp, err := tw.http.Get(u)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}

	for key, value := range resolutions {
		var listURL string

		switch bt {
		case "UPLOAD":
		case "HIGHLIGHT":
		case "ARCHIVE":
		}

		if listURL == "" {
			log.Fatalf("failed to build listURL for vod: %s", vodID)
		}

		if isQualityValid(listURL) {
			vp := &m3u8.VariantPlaylist{
				URL:        listURL,
				Bandwidth:  "", // ????
				Codecs:     "avc1.64002A,mp4a.40.2",
				Resolution: value.Res,
				FrameRate:  value.FPS,
				Video:      key,
			}
			master.Lists = append(master.Lists, vp)
		}
	}

	for _, list := range master.Lists {
		fmt.Println("List:", list)
	}

	return &master, nil
}

func (tw *Client) MasterPlaylistVOD(ctx context.Context, vodID string) ([]byte, error) {
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

	return b, nil
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

// type broadcastType int // ARCHIVE | HIGHLIGHT | CLIP | LIVE
// type videoSort int // TIME | VIEWS
// cursor example: 2732435300|877053|2026-03-26T20:03:29Z|24387
func (tw *Client) FilterableVideoTower_Videos(
	ctx context.Context,
	channel string,
	limit int,
) (*FilterableVideoTower_Videos, error) {
	if limit > 100 {
		return nil, errors.New("limit value must be between 1 and 100")
	}

	gqlPl := `{
        "operationName": "FilterableVideoTower_Videos",
        "variables": {
            "includePreviewBlur": false,
            "limit": %d,
            "channelOwnerLogin": "%s",
            "broadcastType": "ARCHIVE",
            "videoSort": "TIME"
        },
        "extensions": {
            "persistedQuery": {
                "version": 1,
                "sha256Hash": "67004f7881e65c297936f32c75246470629557a393788fb5a69d6d9a25a8fd5f"
            }
        }
    }`

	var videos FilterableVideoTower_Videos
	if err := sendGqlLoadAndDecode(ctx, tw.http, &videos, gqlPl, limit, channel); err != nil {
		return nil, err
	}

	return &videos, nil
}

func (tw *Client) querySeekPreviewsURL(ctx context.Context, vodID string) (string, string, error) {
	gqlPl := `{
		"query": "query { video(id: \"%s\") { broadcastType, id, createdAt, seekPreviewsURL, owner { login } } }"
	}`

	var vod Video
	if err := sendGqlLoadAndDecode(ctx, tw.http, &vod, gqlPl, vodID); err != nil {
		return "", "", err
	}

	return vod.BroadcastType, vod.SeekPreviewsURL, nil
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
