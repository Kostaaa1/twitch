package twitch

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

func (api *API) extractClipSourceURL(videoQualities []VideoQuality, quality string) string {
	fmt.Println(videoQualities)
	if quality == "best" {
		return videoQualities[0].SourceURL
	}
	if quality == "worst" {
		return videoQualities[len(videoQualities)-1].SourceURL
	}
	for _, q := range videoQualities {
		if strings.HasPrefix(quality, q.Quality) || strings.HasPrefix(q.Quality, quality) {
			return q.SourceURL
		}
	}
	return quality
}

func (api *API) GetClipUsherURL(clip Clip, sourceURL string) (string, error) {
	fmt.Println("\nSIG: ", clip.PlaybackAccessToken.Signature)
	fmt.Println("\nTOKEN: ", clip.PlaybackAccessToken.Value)
	URL := fmt.Sprintf("%s?sig=%s&token=%s", sourceURL, url.QueryEscape(clip.PlaybackAccessToken.Signature), url.QueryEscape(clip.PlaybackAccessToken.Value))
	return URL, nil
}

func (api *API) DownloadClip(unit MediaUnit) error {
	clip, err := api.ClipData(unit.Slug)
	if err != nil {
		return err
	}

	sourceURL := api.extractClipSourceURL(clip.Assets[0].VideoQualities, unit.Quality)
	fmt.Println("sourceURL", sourceURL)

	usherURL, err := api.GetClipUsherURL(clip, sourceURL)
	if err != nil {
		return err
	}

	fmt.Println("usherURL\n", usherURL)

	n, err := api.downloadAndWriteSegment(usherURL, unit.W)
	if err != nil {
		return err
	}

	if file, ok := unit.W.(*os.File); ok {
		api.progressCh <- ProgresbarChanData{
			Text:  file.Name(),
			Bytes: n,
		}
	}

	return nil
}

type VideoQuality struct {
	FrameRate float64 `json:"frameRate"`
	Quality   string  `json:"quality"`
	SourceURL string  `json:"sourceURL"`
	Typename  string  `json:"__typename"`
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
	ThumbnailURL        string      `json:"thumbnailURL"`
	CreatedAt           time.Time   `json:"createdAt"`
	IsPublished         bool        `json:"isPublished"`
	DurationSeconds     int         `json:"durationSeconds"`
	ChampBadge          interface{} `json:"champBadge"`
	PlaybackAccessToken struct {
		Signature string `json:"signature"`
		Value     string `json:"value"`
		Typename  string `json:"__typename"`
	} `json:"playbackAccessToken"`
	Video struct {
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
