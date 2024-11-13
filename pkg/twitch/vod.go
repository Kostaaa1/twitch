package twitch

import (
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
)

func (api *API) truncateSegments(mediaPlaylist []byte, start, end time.Duration) ([]string, error) {
	var segmentDuration float64 = 10
	s := int(start.Seconds()/segmentDuration) * 2
	e := int(end.Seconds()/segmentDuration) * 2

	var segments []string
	lines := strings.Split(string(mediaPlaylist), "\n")[8:]

	if s > len(lines) || e > len(lines) {
		return segments, fmt.Errorf("invalid start/end parameters. You've choosen %.2f/%.2f but the video duration is %d", start.Seconds(), end.Seconds(), len(lines)) // TODO: calculate duration properly
	}

	if e == 0 {
		segments = lines[s:]
	} else {
		segments = lines[s:e]
	}
	return segments, nil
}

func (api *API) GetVODMediaPlaylist(slug, quality string) (string, error) {
	if slug == "" {
		return "", fmt.Errorf("slug is required for vod media list")
	}

	master, status, err := api.GetVODMasterM3u8(slug)
	if err != nil && status != http.StatusForbidden {
		return "", err
	}

	var vodPlaylistURL string

	if status == http.StatusForbidden {
		subUrl, err := api.getSubVODPlaylistURL(slug, quality)
		if err != nil {
			return "", err
		}
		vodPlaylistURL = subUrl
	} else {
		variantList, err := master.GetVariantPlaylistByQuality(quality)
		if err != nil {
			return "", err
		}
		vodPlaylistURL = variantList.URL
	}

	return vodPlaylistURL, nil
}

func segmentFileName(segmentURL string) string {
	parts := strings.Split(segmentURL, "/")
	return parts[len(parts)-1]
}

func (api *API) ParallelVodDownload(unit MediaUnit) error {
	vodPlaylistURL, err := api.GetVODMediaPlaylist(unit.Slug, unit.Quality)
	if err != nil {
		return err
	}

	mediaPlaylist, err := api.fetch(vodPlaylistURL)
	if err != nil {
		return err
	}

	segments, err := api.truncateSegments(mediaPlaylist, unit.Start, unit.End)
	if err != nil {
		return err
	}

	tempDir, _ := os.MkdirTemp("", fmt.Sprintf("vod_segments_%s", unit.Slug))
	defer os.RemoveAll(tempDir)

	maxConcurrency := runtime.NumCPU() / 2
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, seg := range segments {
		if strings.HasSuffix(seg, ".ts") {
			wg.Add(1)

			go func(seg string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				if err := api.downloadSegmentToTempFile(seg, vodPlaylistURL, tempDir, unit); err != nil {
					fmt.Println(err)
				}
			}(seg)
		}
	}

	wg.Wait()

	if err := api.writeSegmentsToOutput(tempDir, segments, unit); err != nil {
		return err
	}

	return nil
}

func (api *API) writeSegmentsToOutput(tempDir string, segments []string, unit MediaUnit) error {
	for _, tsFile := range segments {
		if !strings.HasSuffix(tsFile, ".ts") {
			continue
		}
		tempFilePath := fmt.Sprintf("%s/%s", tempDir, segmentFileName(tsFile))
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

// Stream segment, one by one to writer. used in web.
func (api *API) StreamVOD(unit MediaUnit) error {
	vodPlaylistURL, err := api.GetVODMediaPlaylist(unit.Slug, unit.Quality)
	if err != nil {
		return err
	}

	mediaPlaylist, err := api.fetch(vodPlaylistURL)
	if err != nil {
		return err
	}

	segments, err := api.truncateSegments(mediaPlaylist, unit.Start, unit.End)
	if err != nil {
		return err
	}

	for _, segment := range segments {
		if strings.HasSuffix(segment, ".ts") {
			lastIndex := strings.LastIndex(vodPlaylistURL, "/")
			segmentURL := fmt.Sprintf("%s/%s", vodPlaylistURL[:lastIndex], segment)

			n, err := api.downloadAndWriteSegment(segmentURL, unit.W)
			if err != nil {
				fmt.Printf("error downloading segment %s: %v\n", segmentURL, err)
				return err
			}

			if file, ok := unit.W.(*os.File); ok && file != nil {
				api.progressCh <- ProgresbarChanData{
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

func (api *API) GetVODMasterM3u8(slug string) (*m3u8.MasterPlaylist, int, error) {
	token, sig, err := api.getVideoCredentials(slug)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	u := fmt.Sprintf("%s/vod/%s?nauth=%s&nauthsig=%s&allow_audio_only=true&allow_source=true", api.usherURL, slug, token, sig)

	b, code, err := api.fetchWithCode(u)
	if err != nil {
		return nil, code, err
	}

	return m3u8.Master(b), code, nil
}

// Getting the sub VOD playlist
func (api *API) getSubVODPlaylistURL(slug, quality string) (string, error) {
	gqlPayload := `{
 	   "query": "query { video(id: \"%s\") { broadcastType, createdAt, seekPreviewsURL, owner { login } } }"
	}`
	body := strings.NewReader(fmt.Sprintf(gqlPayload, slug))

	var subVodResponse struct {
		Data struct {
			Video struct {
				BroadcastType string    `json:"broadcastType"`
				CreatedAt     time.Time `json:"createdAt"`
				Owner         struct {
					Login string `json:"login"`
				} `json:"owner"`
				SeekPreviewsURL string `json:"seekPreviewsURL"`
			} `json:"video"`
		} `json:"data"`
		Extensions struct {
			DurationMilliseconds int    `json:"durationMilliseconds"`
			RequestID            string `json:"requestID"`
		} `json:"extensions"`
	}

	if err := api.sendGqlLoadAndDecode(body, &subVodResponse); err != nil {
		return "", err
	}

	previewURL, err := url.Parse(subVodResponse.Data.Video.SeekPreviewsURL)
	if err != nil {
		return "", err
	}

	paths := strings.Split(previewURL.Path, "/")
	var vodId string
	for i, p := range paths {
		if p == "storyboards" {
			vodId = paths[i-1]
		}
	}

	// [NOT TESTED]
	// This method works for older uploaded VODS
	// days_difference - difference between current date and p.Data.Video.CreatedAt
	// if broadcastType == "upload" && days_difference > 7 {
	// url = fmt.Sprintf(`https://${domain}/${channelData.login}/${vodId}/${vodSpecialID}/${resKey}/index-dvr.m3u8`, previewURL.Host, p.Data.Video.Owner.Login, slug, vodId, resolution)
	// }
	// resolution := getResolution(quality, v)

	broadcastType := strings.ToLower(subVodResponse.Data.Video.BroadcastType)

	var url string
	if broadcastType == "highlight" {
		url = fmt.Sprintf(`https://%s/%s/%s/highlight-%s.m3u8`, previewURL.Host, vodId, quality, slug)
	} else if broadcastType != "upload" {
		url = fmt.Sprintf(`https://%s/%s/%s/index-dvr.m3u8`, previewURL.Host, vodId, quality)
	}

	return url, nil
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
