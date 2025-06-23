package twitch

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/m3u8"
)

type VideoCredResponse struct {
	Typename  string `json:"__typename"`
	Signature string `json:"signature"`
	Value     string `json:"value"`
}

func (tw *Client) vodTokenAndSignature(id string) (string, string, error) {
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
			VideoPlaybackAccessToken VideoCredResponse `json:"videoPlaybackAccessToken"`
		} `json:"data"`
	}
	var p payload

	if err := tw.sendGqlLoadAndDecode(body, &p); err != nil {
		return "", "", err
	}

	if p.Data.VideoPlaybackAccessToken.Value == "" && p.Data.VideoPlaybackAccessToken.Signature == "" {
		return "", "", fmt.Errorf("[VOD expired] sorry. Unless you've got a time machine, that content is unavailable")
	}

	return p.Data.VideoPlaybackAccessToken.Value, p.Data.VideoPlaybackAccessToken.Signature, nil
}

func (tw *Client) MasterPlaylistVOD(vodID string) (*m3u8.MasterPlaylist, error) {
	token, sig, err := tw.vodTokenAndSignature(vodID)
	if err != nil {
		return nil, err
	}

	m3u8Url := fmt.Sprintf("%s/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true", usherURL, vodID, token, sig)

	b, code, err := tw.fetchWithCode(m3u8Url)
	if code == http.StatusForbidden {
		// this means that you need to be subscribed to access the m3u8 master. In that case, creating fake playlist.
		subVOD, err := tw.SubVodData(vodID)
		if err != nil {
			return nil, err
		}
		previewURL, err := url.Parse(subVOD.Video.SeekPreviewsURL)
		if err != nil {
			return nil, err
		}
		return m3u8.MasterPlaylistMock(tw.httpClient, vodID, previewURL, subVOD.Video.BroadcastType), nil
	}

	if err != nil {
		return nil, err
	}

	return m3u8.Master(b), nil
}

func (tw *Client) FetchAndParseMediaPlaylist(variant m3u8.VariantPlaylist) (*m3u8.MediaPlaylist, error) {
	b, err := tw.fetch(variant.URL)
	if err != nil {
		return nil, err
	}
	parsed := m3u8.ParseMediaPlaylist(b)
	return &parsed, nil
}

type VideoMetadata struct {
	User struct {
		ID              string `json:"id"`
		PrimaryColorHex string `json:"primaryColorHex"`
		IsPartner       bool   `json:"isPartner"`
		ProfileImageURL string `json:"profileImageURL"`
		LastBroadcast   struct {
			ID        string    `json:"id"`
			StartedAt time.Time `json:"startedAt"`
			Typename  string    `json:"__typename"`
		} `json:"lastBroadcast"`
		Stream    any `json:"stream"`
		Followers struct {
			TotalCount int    `json:"totalCount"`
			Typename   string `json:"__typename"`
		} `json:"followers"`
		Typename string `json:"__typename"`
	} `json:"user"`
	CurrentUser any   `json:"currentUser"`
	Video       Video `json:"video"`
}

func (tw *Client) VideoMetadata(id string) (VideoMetadata, error) {
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
	if err := tw.sendGqlLoadAndDecode(body, &p); err != nil {
		return VideoMetadata{}, err
	}

	if p.Data.Video.ID == "" {
		return VideoMetadata{}, fmt.Errorf("failed to get the video data for %s", id)
	}

	return p.Data, nil
}

type Video struct {
	ID                  string        `json:"id"`
	Title               string        `json:"title"`
	PreviewThumbnailURL string        `json:"previewThumbnailURL"`
	PublishedAt         time.Time     `json:"publishedAt"`
	ViewCount           int64         `json:"viewCount"`
	LengthSeconds       int64         `json:"lengthSeconds"`
	AnimatedPreviewURL  string        `json:"animatedPreviewURL"`
	ContentTags         []interface{} `json:"contentTags"`
	CreatedAt           time.Time     `json:"created_at"`
	Self                struct {
		IsRestricted   bool `json:"isRestricted"`
		ViewingHistory struct {
			Position int    `json:"position"`
			Typename string `json:"__typename"`
		} `json:"viewingHistory"`
	} `json:"self"`
	Game struct {
		ID          string `json:"id"`
		Slug        string `json:"slug"`
		BoxArtURL   string `json:"boxArtURL"`
		DisplayName string `json:"displayName"`
		Name        string `json:"name"`
	} `json:"game"`
	Owner struct {
		ID              string `json:"id"`
		DisplayName     string `json:"displayName"`
		Login           string `json:"login"`
		ProfileImageURL string `json:"profileImageURL"`
		PrimaryColorHex string `json:"primaryColorHex"`
	} `json:"owner"`
}

type FilterableVideoTower_Videos struct {
	Data struct {
		User struct {
			ID     string `json:"id"`
			Videos struct {
				Edges []struct {
					Cursor   interface{} `json:"cursor"`
					Node     Video       `json:"node"`
					Typename string      `json:"__typename"`
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
}

func (tw *Client) GetVideosByChannelName(channelName string, limit int) ([]Video, error) {
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
						Cursor   interface{} `json:"cursor"`
						Node     Video       `json:"node"`
						Typename string      `json:"__typename"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"videos"`
			} `json:"user"`
		} `json:"data"`
	}

	var p data
	if err := tw.sendGqlLoadAndDecode(body, &p); err != nil {
		return nil, err
	}

	videos := make([]Video, len(p.Data.User.Videos.Edges))
	for i, video := range p.Data.User.Videos.Edges {
		videos[i] = video.Node
	}

	return videos, nil
}

type SubVODResponse struct {
	Video struct {
		BroadcastType string    `json:"broadcastType"`
		CreatedAt     time.Time `json:"createdAt"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
		SeekPreviewsURL string `json:"seekPreviewsURL"`
	} `json:"video"`
}

func (tw *Client) SubVodData(vodID string) (SubVODResponse, error) {
	gqlPayload := `{
 	   "query": "query { video(id: \"%s\") { broadcastType, createdAt, seekPreviewsURL, owner { login } } }"
	}`
	body := strings.NewReader(fmt.Sprintf(gqlPayload, vodID))

	var subVodResponse struct {
		Data SubVODResponse `json:"data"`
	}
	if err := tw.sendGqlLoadAndDecode(body, &subVodResponse); err != nil {
		return SubVODResponse{}, err
	}
	return subVodResponse.Data, nil
}
