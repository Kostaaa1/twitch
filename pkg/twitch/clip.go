package twitch

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type VideoQualities []struct {
	Typename  string  `json:"__typename"`
	FrameRate float64 `json:"frameRate"`
	Quality   string  `json:"quality"`
	SourceURL string  `json:"sourceURL"`
}

func (api *API) extractClipSourceURL(videoQualities VideoQualities, quality string) string {
	if quality == "best" {
		return videoQualities[0].SourceURL
	}
	if quality == "worst" {
		return videoQualities[len(videoQualities)-1].SourceURL
	}
	for _, q := range videoQualities {
		if strings.HasPrefix(q.Quality, quality) {
			return q.SourceURL
		}
	}
	return quality
}

type ClipCredentials struct {
	Typename            string `json:"__typename"`
	ID                  string `json:"id"`
	PlaybackAccessToken struct {
		Typename  string `json:"__typename"`
		Signature string `json:"signature"`
		Value     string `json:"value"`
	} `json:"playbackAccessToken"`
	VideoQualities VideoQualities `json:"videoQualities"`
}

func (api *API) GetClipData(slug string) (ClipCredentials, error) {
	gqlPayload := `{
        "operationName": "VideoAccessToken_Clip",
        "variables": {
            "slug": "%s"
        },
        "extensions": {
            "persistedQuery": {
                "version": 1,
                "sha256Hash": "36b89d2507fce29e5ca551df756d27c1cfe079e2609642b4390aa4c35796eb11"
            }
        }
    }`

	type payload struct {
		Data struct {
			Clip ClipCredentials `json:"clip"`
		} `json:"data"`
	}
	var p payload

	body := strings.NewReader(fmt.Sprintf(gqlPayload, slug))
	if err := api.sendGqlLoadAndDecode(body, &p); err != nil {
		return ClipCredentials{}, err
	}

	return p.Data.Clip, nil
}

// returns the URL. When fetched, returns clip mp4 bytes
func (api *API) GetClipUsherURL(slug, quality string) (string, error) {
	clip, err := api.GetClipData(slug)
	if err != nil {
		return "", err
	}

	sourceURL := api.extractClipSourceURL(clip.VideoQualities, quality)
	URL := fmt.Sprintf("%s?sig=%s&token=%s", sourceURL, url.QueryEscape(clip.PlaybackAccessToken.Signature), url.QueryEscape(clip.PlaybackAccessToken.Value))
	return URL, nil
}

func (api *API) downloadClip(unit MediaUnit) error {
	usherURL, err := api.GetClipUsherURL(unit.Slug, unit.Quality)
	if err != nil {
		return err
	}

	n, err := api.downloadAndWriteSegment(usherURL, unit.File)
	if err != nil {
		return err
	}

	api.progressCh <- ProgresbarChanData{
		Text:  unit.File.Name(),
		Bytes: n,
	}

	return nil
}

type Clip struct {
	Broadcaster struct {
		Typename    string `json:"__typename"`
		DisplayName string `json:"displayName"`
		Followers   struct {
			Typename   string `json:"__typename"`
			TotalCount int    `json:"totalCount"`
		} `json:"followers"`
		ID            string `json:"id"`
		IsPartner     bool   `json:"isPartner"`
		LastBroadcast struct {
			Typename  string    `json:"__typename"`
			ID        string    `json:"id"`
			StartedAt time.Time `json:"startedAt"`
		} `json:"lastBroadcast"`
		Login           string `json:"login"`
		PrimaryColorHex string `json:"primaryColorHex"`
		ProfileImageURL string `json:"profileImageURL"`
		Self            any    `json:"self"`
		Stream          any    `json:"stream"`
	} `json:"broadcaster"`
	ChampBadge any       `json:"champBadge"`
	CreatedAt  time.Time `json:"createdAt"`
	Curator    struct {
		Typename        string `json:"__typename"`
		DisplayName     string `json:"displayName"`
		ID              string `json:"id"`
		Login           string `json:"login"`
		ProfileImageURL string `json:"profileImageURL"`
	} `json:"curator"`
	DurationSeconds int    `json:"durationSeconds"`
	EmbedURL        string `json:"embedURL"`
	Game            struct {
		Typename    string `json:"__typename"`
		BoxArtURL   string `json:"boxArtURL"`
		DisplayName string `json:"displayName"`
		ID          string `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
	} `json:"game"`
	ID                     string `json:"id"`
	IsFeatured             bool   `json:"isFeatured"`
	IsPublished            bool   `json:"isPublished"`
	IsViewerEditRestricted bool   `json:"isViewerEditRestricted"`
	Language               string `json:"language"`
	PlaybackAccessToken    struct {
		Typename  string `json:"__typename"`
		Signature string `json:"signature"`
		Value     string `json:"value"`
	} `json:"playbackAccessToken"`
	Slug              string `json:"slug"`
	SuggestedCropping any    `json:"suggestedCropping"`
	ThumbnailURL      string `json:"thumbnailURL"`
	Title             string `json:"title"`
	URL               string `json:"url"`
	Video             struct {
		Typename      string `json:"__typename"`
		BroadcastType string `json:"broadcastType"`
		ID            string `json:"id"`
		Title         string `json:"title"`
	} `json:"video"`
	VideoOffsetSeconds int `json:"videoOffsetSeconds"`
	VideoQualities     []struct {
		Typename  string `json:"__typename"`
		SourceURL string `json:"sourceURL"`
	} `json:"videoQualities"`
	ViewCount int `json:"viewCount"`
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

	bss, _ := json.MarshalIndent(result, "", " ")
	fmt.Println(string(bss))

	return result.Data.Clip, nil
}
