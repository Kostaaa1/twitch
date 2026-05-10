package twitch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch/m3u8"
)

func (tw *Client) vodTokenAndSignature(ctx context.Context, id string) (string, string, error) {
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
			VideoPlaybackAccessToken VideoPlaybackAccessToken `json:"videoPlaybackAccessToken"`
		} `json:"data"`
	}
	var p payload

	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
		return "", "", err
	}

	if p.Data.VideoPlaybackAccessToken.Value == "" && p.Data.VideoPlaybackAccessToken.Signature == "" {
		return "", "", fmt.Errorf("[VOD expired] sorry. Unless you've got a time machine, that content is unavailable")
	}

	return p.Data.VideoPlaybackAccessToken.Value, p.Data.VideoPlaybackAccessToken.Signature, nil
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

func (tw *Client) PlaybackAccessToken(ctx context.Context, login string) error {
	gqlPayload := `{
	    "operationName": "PlaybackAccessToken_Template",
	    "query": "query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isLive) {    value    signature   authorization { isForbidden forbiddenReasonCode }   __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isVod) {    value    signature   __typename  }}",
	    "variables": {
			"isLive": true,
			"login": "%s",
			"isVod": false,
			"vodID": "",
			"playerType": "site",
			"platform": "web"
		}
	}`

	body := strings.NewReader(fmt.Sprintf(gqlPayload, login))
	type payload struct {
		Data struct {
			VideoPlaybackAccessToken VideoPlaybackAccessToken `json:"videoPlaybackAccessToken"`
		} `json:"data"`
	}
	var p payload

	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
		return err
	}

	return nil
}

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
	value, sig, err := tw.vodTokenAndSignature(ctx, vodID)
	if err != nil {
		return nil, err
	}

	m3u8Url := fmt.Sprintf("%s/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true", usherURL, vodID, value, sig)

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

// type FilterableVideoTower_Videos struct {
// 	Data struct {
// 		User struct {
// 			ID     string `json:"id"`
// 			Videos struct {
// 				Edges []struct {
// 					Cursor interface{} `json:"cursor"`
// 					Node   Video       `json:"node"`
// 				} `json:"edges"`
// 				PageInfo struct {
// 					HasNextPage bool `json:"hasNextPage"`
// 				} `json:"pageInfo"`
// 			} `json:"videos"`
// 		} `json:"user"`
// 	} `json:"data"`
// }

type FilterableVideoTower_Videos []struct {
	Data struct {
		User struct {
			ID     string `json:"id"`
			Videos struct {
				Edges []struct {
					Cursor time.Time `json:"cursor"`
					Node   struct {
						AnimatedPreviewURL string `json:"animatedPreviewURL"`
						Game               struct {
							BoxArtURL   string `json:"boxArtURL"`
							ID          string `json:"id"`
							Slug        string `json:"slug"`
							DisplayName string `json:"displayName"`
							Name        string `json:"name"`
							Typename    string `json:"__typename"`
						} `json:"game"`
						BroadcastIdentifier struct {
							ID       string `json:"id"`
							Typename string `json:"__typename"`
						} `json:"broadcastIdentifier"`
						ID            string `json:"id"`
						LengthSeconds int    `json:"lengthSeconds"`
						Owner         struct {
							DisplayName     string `json:"displayName"`
							ID              string `json:"id"`
							Login           string `json:"login"`
							ProfileImageURL string `json:"profileImageURL"`
							PrimaryColorHex any    `json:"primaryColorHex"`
							Roles           struct {
								IsPartner bool   `json:"isPartner"`
								Typename  string `json:"__typename"`
							} `json:"roles"`
							Typename string `json:"__typename"`
						} `json:"owner"`
						PreviewThumbnailURL string    `json:"previewThumbnailURL"`
						PublishedAt         time.Time `json:"publishedAt"`
						Self                struct {
							IsRestricted   bool `json:"isRestricted"`
							ViewingHistory struct {
								Position  int       `json:"position"`
								UpdatedAt time.Time `json:"updatedAt"`
								Typename  string    `json:"__typename"`
							} `json:"viewingHistory"`
							Typename string `json:"__typename"`
						} `json:"self"`
						Title               string `json:"title"`
						ViewCount           int    `json:"viewCount"`
						ResourceRestriction any    `json:"resourceRestriction"`
						ContentTags         []any  `json:"contentTags"`
						Typename            string `json:"__typename"`
					} `json:"node"`
					Typename string `json:"__typename"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					Typename    string `json:"__typename"`
				} `json:"pageInfo"`
				Typename string `json:"__typename"`
			} `json:"videos"`
			Typename string `json:"__typename"`
		} `json:"user"`
	} `json:"data"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}

func (tw *Client) ListVideosByChannelName(ctx context.Context, channelName string, limit int) ([]Video, error) {
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

	body := strings.NewReader(fmt.Sprintf(gqlPl, limit, channelName))

	type data struct {
		Data struct {
			User struct {
				ID     string `json:"id"`
				Videos struct {
					Edges []struct {
						Cursor interface{} `json:"cursor"`
						Node   Video       `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"videos"`
			} `json:"user"`
		} `json:"data"`
	}

	var p data
	if err := tw.sendGqlLoadAndDecode(ctx, body, &p); err != nil {
		return nil, err
	}

	videos := make([]Video, len(p.Data.User.Videos.Edges))
	for i, video := range p.Data.User.Videos.Edges {
		videos[i] = video.Node
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
