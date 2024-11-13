package twitch

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (api *API) GetClipUsherURL(clip PlaybackAccessToken, sourceURL string) (string, error) {
	URL := fmt.Sprintf("%s?sig=%s&token=%s", sourceURL, url.QueryEscape(clip.Signature), url.QueryEscape(clip.Value))
	return URL, nil
}

func (api *API) DownloadClip(unit MediaUnit) error {
	clip, err := api.ClipData(unit.Slug)
	if err != nil {
		return err
	}

	sourceURL := extractClipSourceURL(clip.Assets[0].VideoQualities, unit.Quality)
	usherURL, err := api.GetClipUsherURL(clip.PlaybackAccessToken, sourceURL)
	if err != nil {
		return err
	}

	var writtenBytes int64
	if api.config.Downloader.IsFFmpegEnabled && unit.Quality == "audio_only" {
		writtenBytes, err = extractAudio(usherURL, unit.W)
	} else {
		writtenBytes, err = api.downloadAndWriteSegment(usherURL, unit.W)
	}

	if err != nil {
		return err
	}

	if file, ok := unit.W.(*os.File); ok && file != nil {
		api.progressCh <- ProgresbarChanData{
			Text:  file.Name(),
			Bytes: writtenBytes,
		}
	}

	return nil
}

func extractAudio(segmentURL string, output io.Writer) (int64, error) {
	cmd := exec.Command("ffmpeg", "-i", segmentURL, "-q:a", "0", "-map", "a", "-f", "mp3", "-")
	cmd.Stdout = nil
	cmd.Stderr = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	n, err := io.Copy(output, stdout)
	if err != nil {
		return 0, fmt.Errorf("failed to copy audio data: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return 0, fmt.Errorf("FFmpeg conversion failed: %w", err)
	}

	return n, nil
}

type VideoQuality struct {
	FrameRate float64 `json:"frameRate"`
	Quality   string  `json:"quality"`
	SourceURL string  `json:"sourceURL"`
	Typename  string  `json:"__typename"`
}

type PlaybackAccessToken struct {
	Signature string `json:"signature"`
	Value     string `json:"value"`
	Typename  string `json:"__typename"`
}

type ClipAccessToken struct {
	ID                  string              `json:"id"`
	PlaybackAccessToken PlaybackAccessToken `json:"playbackAccessToken"`
	VideoQualities      []VideoQuality      `json:"videoQualities"`
}

type Clip struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	URL        string `json:"url"`
	EmbedURL   string `json:"embedURL"`
	Title      string `json:"title"`
	ViewCount  int64  `json:"viewCount"`
	Language   string `json:"language"`
	IsFeatured bool   `json:"isFeatured"`
	Assets     []struct {
		ID            string    `json:"id"`
		AspectRatio   float64   `json:"aspectRatio"`
		Type          string    `json:"type"`
		CreatedAt     time.Time `json:"createdAt"`
		CreationState string    `json:"creationState"`
		Curator       struct {
			ID              string `json:"id"`
			Login           string `json:"login"`
			DisplayName     string `json:"displayName"`
			ProfileImageURL string `json:"profileImageURL"`
			Typename        string `json:"__typename"`
		} `json:"curator"`
		ThumbnailURL     string         `json:"thumbnailURL"`
		VideoQualities   []VideoQuality `json:"videoQualities"`
		PortraitMetadata interface{}    `json:"portraitMetadata"`
		Typename         string         `json:"__typename"`
	} `json:"assets"`
	Curator struct {
		ID              string `json:"id"`
		Login           string `json:"login"`
		DisplayName     string `json:"displayName"`
		ProfileImageURL string `json:"profileImageURL"`
		Typename        string `json:"__typename"`
	} `json:"curator"`
	Game struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		BoxArtURL   string `json:"boxArtURL"`
		DisplayName string `json:"displayName"`
		Slug        string `json:"slug"`
		Typename    string `json:"__typename"`
	} `json:"game"`
	Broadcast struct {
		ID       string      `json:"id"`
		Title    interface{} `json:"title"`
		Typename string      `json:"__typename"`
	} `json:"broadcast"`
	Broadcaster struct {
		ID              string `json:"id"`
		Login           string `json:"login"`
		DisplayName     string `json:"displayName"`
		PrimaryColorHex string `json:"primaryColorHex"`
		IsPartner       bool   `json:"isPartner"`
		ProfileImageURL string `json:"profileImageURL"`
		Followers       struct {
			TotalCount int    `json:"totalCount"`
			Typename   string `json:"__typename"`
		} `json:"followers"`
		Stream        interface{} `json:"stream"`
		LastBroadcast struct {
			ID        string    `json:"id"`
			StartedAt time.Time `json:"startedAt"`
			Typename  string    `json:"__typename"`
		} `json:"lastBroadcast"`
		Self struct {
			IsEditor bool   `json:"isEditor"`
			Typename string `json:"__typename"`
		} `json:"self"`
		Typename string `json:"__typename"`
	} `json:"broadcaster"`
	ThumbnailURL        string              `json:"thumbnailURL"`
	CreatedAt           time.Time           `json:"createdAt"`
	IsPublished         bool                `json:"isPublished"`
	DurationSeconds     int                 `json:"durationSeconds"`
	ChampBadge          interface{}         `json:"champBadge"`
	PlaybackAccessToken PlaybackAccessToken `json:"playbackAccessToken"`
	Video               struct {
		ID            string `json:"id"`
		BroadcastType string `json:"broadcastType"`
		Title         string `json:"title"`
		Typename      string `json:"__typename"`
	} `json:"video"`
	VideoOffsetSeconds int `json:"videoOffsetSeconds"`
	VideoQualities     []struct {
		SourceURL string `json:"sourceURL"`
		Typename  string `json:"__typename"`
	} `json:"videoQualities"`
	IsViewerEditRestricted bool   `json:"isViewerEditRestricted"`
	Typename               string `json:"__typename"`
}

func (api *API) ClipData(slug string) (Clip, error) {
	gqlPayload := `{
        "operationName": "ShareClipRenderStatus",
        "variables": {
            "slug": "%s"
        },
        "extensions": {
            "persistedQuery": {
                "version": 1,
                "sha256Hash": "f130048a462a0ac86bb54d653c968c514e9ab9ca94db52368c1179e97b0f16eb"
            }
        }
    }`

	var result struct {
		Data struct {
			Clip Clip `json:"clip"`
		} `json:"data"`
	}

	body := strings.NewReader(fmt.Sprintf(gqlPayload, slug))
	if err := api.sendGqlLoadAndDecode(body, &result); err != nil {
		return Clip{}, err
	}

	if result.Data.Clip.ID == "" {
		return Clip{}, fmt.Errorf("failed to get the video data for %s", slug)
	}

	return result.Data.Clip, nil
}
