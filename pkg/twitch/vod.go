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

func (api *API) getVideoCredentials(id string) (string, string, error) {
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

	if err := api.sendGqlLoadAndDecode(body, &p); err != nil {
		return "", "", err
	}

	if p.Data.VideoPlaybackAccessToken.Value == "" && p.Data.VideoPlaybackAccessToken.Signature == "" {
		return "", "", fmt.Errorf("sorry. Unless you've got a time machine, that content is unavailable")
	}

	return p.Data.VideoPlaybackAccessToken.Value, p.Data.VideoPlaybackAccessToken.Signature, nil
}

func (api *API) GetVODMasterM3u8(vodID string) (*m3u8.MasterPlaylist, int, error) {
	token, sig, err := api.getVideoCredentials(vodID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	m3u8Url := fmt.Sprintf("%s/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true", usherURL, vodID, token, sig)

	b, code, err := api.fetchWithCode(m3u8Url)
	if code == http.StatusForbidden {
		// this means that you need to be subscribed to access the m3u8 master. In that case, creating fake playlist.
		subVOD, err := api.SubVODData(vodID)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		previewURL, err := url.Parse(subVOD.Video.SeekPreviewsURL)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		return m3u8.CreateFakeMaster(api.client, vodID, previewURL, subVOD.Video.BroadcastType), http.StatusOK, nil
	}

	if err != nil {
		return nil, code, err
	}

	return m3u8.Master(b), code, nil
}

func (api *API) GetVODMediaPlaylist(variant m3u8.VariantPlaylist) (*m3u8.MediaPlaylist, error) {
	mediaPlaylist, err := api.fetch(variant.URL)
	if err != nil {
		return nil, err
	}
	parsed := m3u8.ParseMediaPlaylist(mediaPlaylist)
	return &parsed, nil
}

// Getting the sub VOD playlist
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

func (api *API) VideoMetadata(id string) (VideoMetadata, error) {
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
	if err := api.sendGqlLoadAndDecode(body, &p); err != nil {
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
	ResourceRestriction interface{}   `json:"resourceRestriction"`
	ContentTags         []interface{} `json:"contentTags"`
	CreatedAt           time.Time     `json:"created_at"`
	Self                struct {
		IsRestricted   bool `json:"isRestricted"`
		ViewingHistory struct {
			Position int    `json:"position"`
			Typename string `json:"__typename"`
		} `json:"viewingHistory"`
		Typename string `json:"__typename"`
	} `json:"self"`
	Game struct {
		ID          string `json:"id"`
		Slug        string `json:"slug"`
		BoxArtURL   string `json:"boxArtURL"`
		DisplayName string `json:"displayName"`
		Name        string `json:"name"`
		Typename    string `json:"__typename"`
	} `json:"game"`
	Owner struct {
		ID              string `json:"id"`
		DisplayName     string `json:"displayName"`
		Login           string `json:"login"`
		ProfileImageURL string `json:"profileImageURL"`
		PrimaryColorHex string `json:"primaryColorHex"`
		Typename        string `json:"__typename"`
	} `json:"owner"`
	Typename string `json:"__typename"`
}

func (api *API) GetVideosByUsername(username string) ([]Video, error) {
	gqlPl := `{
		"operationName": "HomeShelfVideos",
		"variables": {
			"channelLogin": "%s",
			"first": 1
		},
		"extensions": {
			"persistedQuery": {
				"version": 1,
				"sha256Hash": "951c268434dc36a482c6f854215df953cf180fc2757f1e0e47aa9821258debf7"
			}
		}
	}`

	body := strings.NewReader(fmt.Sprintf(gqlPl, username))

	type payload struct {
		Data struct {
			User struct {
				ID           string `json:"id"`
				VideoShelves struct {
					Edges []struct {
						Node struct {
							Items    []Video `json:"items"`
							Typename string  `json:"__typename"`
						} `json:"node"`
						Typename string `json:"__typename"`
					} `json:"edges"`
					Typename string `json:"__typename"`
				} `json:"videoShelves"`
				Typename string `json:"__typename"`
			} `json:"user"`
		} `json:"data"`
	}
	var p payload

	if err := api.sendGqlLoadAndDecode(body, &p); err != nil {
		return nil, err
	}

	return p.Data.User.VideoShelves.Edges[0].Node.Items, nil
}
