package twitch

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

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

type Curator struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"displayName"`
	ProfileImageURL string `json:"profileImageURL"`
	Typename        string `json:"__typename"`
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
	Curator Curator `json:"curator"`
	Game    struct {
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

func (c *Client) ConstructUsherURL(clip PlaybackAccessToken, sourceURL string) (string, error) {
	return fmt.Sprintf("%s?sig=%s&token=%s", sourceURL, url.QueryEscape(clip.Signature), url.QueryEscape(clip.Value)), nil
}

func (c *Client) ClipMetadata(slug string) (Clip, error) {
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
	if err := c.sendGqlLoadAndDecode(body, &result); err != nil {
		return Clip{}, err
	}

	if result.Data.Clip.ID == "" {
		return Clip{}, fmt.Errorf("failed to get the clip data for %s", slug)
	}

	return result.Data.Clip, nil
}
