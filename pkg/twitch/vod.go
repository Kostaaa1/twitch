package twitch

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func (tw *Client) mockMasterPlaylist(ctx context.Context, vodID string) (*m3u8.MasterPlaylist, error) {
	return nil, errors.New("not impl")

	// subVOD, err := tw.SubVodData(ctx, vodID)
	// if err != nil {
	// 	return nil, err
	// }

	// bcType := strings.ToLower(subVOD.Video.BroadcastType)

	// previewURL := subVOD.Video.SeekPreviewsURL
	// if previewURL != "" {
	// 	// other way
	// }
	// // d3vd9lfkzbru3h.cloudfront.net

	// if previewURL == "" {
	// 	return nil, fmt.Errorf("failed to acquire previewURL for video: %s", vodID)
	// }

	// parsed, err := url.Parse(previewURL)
	// if err != nil {
	// 	return nil, err
	// }

	// listHost := parsed.Host
	// paths := strings.Split(parsed.Path, "/")
	// var listSourceID string
	// for i, p := range paths {
	// 	if p == "storyboards" {
	// 		listSourceID = paths[i-1]
	// 	}
	// }

	// if listHost == "" || listSourceID == "" {
	// 	return nil, fmt.Errorf("failed to find the host and source ID for mock master playlist: host=%s source=%s", listHost, listSourceID)
	// }

	// master := m3u8.MasterPlaylist{
	// 	Origin: "s3",
	// 	B:      false,
	// 	Region: "EU",
	// 	UserIP: "127.0.0.1",
	// 	// ServingID:       createServingID(),
	// 	Cluster:         "cloudfront_vod",
	// 	UserCountry:     "BE",
	// 	ManifestCluster: "cloudfront_vod",
	// }

	// resolutions := map[string]struct {
	// 	Res string
	// 	FPS string
	// }{
	// 	"chunked":    {Res: "1920x1080", FPS: "60"},
	// 	"720p60":     {Res: "1280x720", FPS: "60"},
	// 	"720p30":     {Res: "1280x720", FPS: "30"},
	// 	"480p30":     {Res: "854x480", FPS: "30"},
	// 	"360p30":     {Res: "640x360", FPS: "30"},
	// 	"160p30":     {Res: "284x160", FPS: "30"},
	// 	"audio_only": {Res: "audio_only", FPS: ""},
	// }

	// isQualityValid := func(u string) bool {
	// 	resp, err := tw.http.Get(u)
	// 	if err != nil {
	// 		return false
	// 	}
	// 	defer resp.Body.Close()
	// 	return resp.StatusCode == http.StatusOK
	// }

	// for key, value := range resolutions {
	// 	var listURL string

	// 	switch bcType {
	// 	case "upload":
	// 	case "highlight":
	// 	case "archive":
	// 	default:
	// 		listURL = fmt.Sprintf(`https://%s/%s/%s/index-dvr.m3u8`, listHost, listSourceID, key)
	// 	}

	// 	if listURL == "" {
	// 		log.Fatalf("failed to build listURL for vod: %s", vodID)
	// 	}

	// 	if isQualityValid(listURL) {
	// 		if key == "chunked" {
	// 			key = "1080p60"
	// 		}
	// 		vp := &m3u8.VariantPlaylist{
	// 			URL:        listURL,
	// 			Bandwidth:  "", // ????
	// 			Codecs:     "avc1.64002A,mp4a.40.2",
	// 			Resolution: value.Res,
	// 			FrameRate:  value.FPS,
	// 			Video:      key,
	// 		}
	// 		master.Lists = append(master.Lists, vp)
	// 	}
	// }

	// return &master, nil
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

// type PreviewParts struct{}

// func (tw *Client) PreviewParts()

func (tw *Client) VideoMetadata(ctx context.Context, vodID string) (VideoMetadata, error) {
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

	body := strings.NewReader(fmt.Sprintf(gqlPayload, vodID))

	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
		return VideoMetadata{}, err
	}

	if p.Data.Video.ID == "" {
		return VideoMetadata{}, fmt.Errorf("failed to get the video data for %s", vodID)
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

type VideoPlaylistURLBuilder struct {
	VodID           string
	Subdomain       string
	Source          string
	BroadcasterType string
	Quality         string
}

func (pup VideoPlaylistURLBuilder) PlaylistURL() string {
	var u string

	switch strings.ToLower(pup.BroadcasterType) {
	case "live":
	case "clip":
	case "premiere":
	case "upload":
	case "highlight":
		u = fmt.Sprintf("https://%s.cloudfront.net/%s/%s/highlight-%s.m3u8", pup.Subdomain, pup.Source, "chunked", pup.VodID)
	case "archive":
		u = fmt.Sprintf("https://%s.cloudfront.net/%s/%s/index-dvr.m3u8", pup.Subdomain, pup.Source, "chunked")
	}

	return u
}

func (pup VideoPlaylistURLBuilder) validate() error {
	// pup.Subdomain
	// pup.Source
	// pup.BroadcasterType
	return nil
}

func (tw *Client) querySeekPreviewsURL(ctx context.Context, vodID string) (string, string, error) {
	gqlPayload := `{
	 	   "query": "query { video(id: \"%s\") { broadcastType, id, createdAt, seekPreviewsURL, owner { login } } }"
		}`

	body := strings.NewReader(fmt.Sprintf(gqlPayload, vodID))

	var vod struct {
		Data struct {
			Video Video `json:"video"`
		} `json:"data"`
	}

	if err := tw.sendGqlLoadAndDecode(ctx, body, &vod); err != nil {
		return "", "", err
	}

	video := vod.Data.Video

	return video.BroadcastType, video.SeekPreviewsURL, nil
}

func (tw *Client) VideoPlaylistBuilder(ctx context.Context, vodID string) (*VideoPlaylistURLBuilder, error) {
	broadcasterType, seekPreviewURL, err := tw.querySeekPreviewsURL(ctx, vodID)
	if err != nil {
		return nil, err
	}

	pup := new(VideoPlaylistURLBuilder)

	if seekPreviewURL != "" {
		u, err := url.Parse(seekPreviewURL)
		if err != nil {
			return nil, err
		}

		subdomainParts := strings.Split(u.Hostname(), ".")
		pup.Subdomain = subdomainParts[0]

		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		pup.Source = parts[0]
		pup.BroadcasterType = broadcasterType
		pup.VodID = vodID
	} else {
		data, err := tw.VideoMetadata(context.Background(), "2766330803")
		if err != nil {
			return nil, err
		}

		parsed, err := url.Parse(data.Video.PreviewThumbnailURL)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(parsed.Path, "/")
		pup.Subdomain = parts[1]
		pup.Source = parts[2]
		pup.BroadcasterType = data.Video.BroadcastType
		pup.VodID = data.Video.ID
	}

	if err := pup.validate(); err != nil {
		return nil, err
	}

	return pup, nil
}
