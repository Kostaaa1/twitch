package twitch

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/internal/m3u8"
	"github.com/Kostaaa1/twitch/internal/spinner"
)

func segmentFileName(segmentURL string) string {
	parts := strings.Split(segmentURL, "/")
	return parts[len(parts)-1]
}

func (api *API) ParallelVodDownload(unit MediaUnit) error {
	if unit.Slug == "" {
		return errors.New("slug is required for vod media list")
	}

	master, status, err := api.GetVODMasterM3u8(unit.Slug)
	if err != nil && status != http.StatusForbidden {
		return err
	}

	variant, err := master.GetVariantPlaylistByQuality(unit.Quality)
	if err != nil {
		return err
	}

	mp, err := api.fetch(variant.URL)
	if err != nil {
		return err
	}

	media := m3u8.ParseMediaPlaylist(string(mp))
	if err := media.TruncateSegments(unit.Start, unit.End); err != nil {
		return err
	}

	tempDir, _ := os.MkdirTemp("", fmt.Sprintf("vod_segments_%s", unit.Slug))
	defer os.RemoveAll(tempDir)

	maxConcurrency := runtime.NumCPU() / 2
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, segURL := range media.Segments {
		if strings.HasSuffix(segURL, ".ts") {
			wg.Add(1)

			go func(segURL string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				if err := api.downloadSegmentToTempFile(segURL, variant.URL, tempDir, unit); err != nil {
					fmt.Println(err)
				}
			}(segURL)
		}
	}

	wg.Wait()
	if err := api.writeSegmentsToOutput(media.Segments, tempDir, unit); err != nil {
		return err
	}

	return nil
}

func (api *API) writeSegmentsToOutput(segments []string, tempDir string, unit MediaUnit) error {
	for _, segURL := range segments {
		if !strings.HasSuffix(segURL, ".ts") {
			continue
		}
		tempFilePath := fmt.Sprintf("%s/%s", tempDir, segmentFileName(segURL))
		tempFile, err := os.Open(tempFilePath)
		if err != nil {
			return fmt.Errorf("failed to open temp file %s: %w", tempFilePath, err)
		}
		if _, err := io.Copy(unit.W, tempFile); err != nil {
			tempFile.Close()
			return fmt.Errorf("failed to write segment to output file: %w", err)
		}
		tempFile.Close()
	}
	return nil
}

func (api *API) StreamVOD(unit MediaUnit) error {
	if unit.Slug == "" {
		return errors.New("slug is required for vod media list")
	}

	master, status, err := api.GetVODMasterM3u8(unit.Slug)
	if err != nil && status != http.StatusForbidden {
		return err
	}

	variant, err := master.GetVariantPlaylistByQuality(unit.Quality)
	if err != nil {
		return err
	}

	mediaPlaylist, err := api.fetch(variant.URL)
	if err != nil {
		return err
	}

	playlist := m3u8.ParseMediaPlaylist(string(mediaPlaylist))
	if err := playlist.TruncateSegments(unit.Start, unit.End); err != nil {
		return err
	}

	for _, segment := range playlist.Segments {
		if strings.HasSuffix(segment, ".ts") {
			lastIndex := strings.LastIndex(variant.URL, "/")
			segmentURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], segment)

			n, err := api.downloadAndWriteSegment(segmentURL, unit.W)
			if err != nil {
				fmt.Printf("error downloading segment %s: %v\n", segmentURL, err)
				return err
			}

			if file, ok := unit.W.(*os.File); ok && file != nil {
				api.progressCh <- spinner.ChannelMessage{
					Text:  file.Name(),
					Bytes: n,
				}
			}
		}
	}

	return nil
}

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
		return m3u8.CreateMockMaster(api.client, vodID, previewURL, subVOD.Video.BroadcastType), http.StatusOK, nil
	}

	if err != nil {
		return nil, code, err
	}

	return m3u8.Master(b), code, nil
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
	CurrentUser any `json:"currentUser"`
	Video       struct {
		ID                  string    `json:"id"`
		Title               string    `json:"title"`
		Description         any       `json:"description"`
		PreviewThumbnailURL string    `json:"previewThumbnailURL"`
		CreatedAt           time.Time `json:"createdAt"`
		ViewCount           int64     `json:"viewCount"`
		PublishedAt         time.Time `json:"publishedAt"`
		LengthSeconds       int64     `json:"lengthSeconds"`
		BroadcastType       string    `json:"broadcastType"`
		Owner               struct {
			ID          string `json:"id"`
			Login       string `json:"login"`
			DisplayName string `json:"displayName"`
			Typename    string `json:"__typename"`
		} `json:"owner"`
		Game struct {
			ID          string `json:"id"`
			Slug        string `json:"slug"`
			BoxArtURL   string `json:"boxArtURL"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Typename    string `json:"__typename"`
		} `json:"game"`
		Typename string `json:"__typename"`
	} `json:"video"`
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
