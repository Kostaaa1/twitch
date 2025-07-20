package kick

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/m3u8"
)

func (c *Client) GetMasterPlaylistURL(channel, vodID string) (string, error) {
	videos, err := c.Videos(channel)
	if err != nil {
		return "", err
	}

	for _, data := range videos {
		if data.Video.UUID == vodID {
			return data.Source, nil
		}
	}

	return "", errors.New("master.m3u8 not found")
}

func (c *Client) GetMediaPlaylist(masterURL, quality string) (*m3u8.MediaPlaylist, error) {
	mediaURL := strings.Replace(masterURL, "master.m3u8", fmt.Sprintf("%s/%s", quality, "playlist.m3u8"), 1)
	fmt.Println(mediaURL)

	resp, err := c.client.Get(mediaURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	playlist := m3u8.ParseMediaPlaylist(b)
	return &playlist, nil
}

func (c *Client) Videos(channel string) ([]*VideoMetadata, error) {
	videosURL := fmt.Sprintf("https://kick.com/api/v2/channels/%s/videos", channel)

	req, err := http.NewRequest(http.MethodGet, videosURL, nil)
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	var videos []*VideoMetadata
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return nil, err
	}

	var channelData *Channel

	for _, video := range videos {
		if video.Source == "" {
			if channelData == nil {
				data, err := c.Channel(channel)
				if err != nil {
					return nil, err
				}
				channelData = data
			}

			if channelData != nil {
				// video.StartTime //
				// video.StartTime.Year()
				// video.StartTime.Year()
				// // https://stream.kick.com/ivs/v1/196233775518/BqIVEMfsiezg/2025/7/18/20/21/Bq7iAn01hQIX/media/hls/master.m3u8
				// // filepath.Join("", channelData.CustomerID, video.StartTime.Year())
				video.Source = fmt.Sprintf("https://stream.kick.com/ivs/v1/%s/%d/%d/%d/%d/%d/%s/media/hls/master.m3u8", channelData.CustomerID, video.StartTime.Year(), video.StartTime.Month(), video.StartTime.Day(), video.StartTime.Hour(), video.StartTime.Minute(), channelData.ContentID)
				fmt.Println("URL: ", video.Source)
			}
		}
	}

	return videos, nil
}

// func (c *Client) handleVodSource(channel string, videos []*VideoMetadata) {
// 	channel, err := c.Channel(channel)
// 	for _, video := range videos {
// 		if video.Source == "" {
// 		}
// 	}
// }

// func getUserVideoIDs() (contentId, customerId string) {
// url := "http://kick.com/api/v2/"
// }

type Video struct {
	CreatedAt         time.Time   `json:"created_at"`
	DeletedAt         interface{} `json:"deleted_at"`
	ID                int         `json:"id"`
	IsPrivate         bool        `json:"is_private"`
	IsPruned          bool        `json:"is_pruned"`
	LiveStreamID      int         `json:"live_stream_id"`
	S3                interface{} `json:"s3"`
	Slug              interface{} `json:"slug"`
	Status            string      `json:"status"`
	Thumb             interface{} `json:"thumb"`
	TradingPlatformID interface{} `json:"trading_platform_id"`
	UpdatedAt         time.Time   `json:"updated_at"`
	UUID              string      `json:"uuid"`
	Views             int         `json:"views"`
}

func (c *Client) VideoByID(id string) (interface{}, error) {
	videoURL := fmt.Sprintf("https://kick.com/api/v1/video/%s", id)

	req, err := http.NewRequest(http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var video Video
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		return nil, err
	}

	return video, nil
}

// Channel
type Channel struct {
	BannerImage struct {
		URL string `json:"url"`
	} `json:"banner_image"`
	CanHost  bool `json:"can_host"`
	Chatroom struct {
		ChannelID            int       `json:"channel_id"`
		ChatMode             string    `json:"chat_mode"`
		ChatModeOld          string    `json:"chat_mode_old"`
		ChatableID           int       `json:"chatable_id"`
		ChatableType         string    `json:"chatable_type"`
		CreatedAt            time.Time `json:"created_at"`
		EmotesMode           bool      `json:"emotes_mode"`
		FollowersMode        bool      `json:"followers_mode"`
		FollowingMinDuration int       `json:"following_min_duration"`
		ID                   int       `json:"id"`
		MessageInterval      int       `json:"message_interval"`
		SlowMode             bool      `json:"slow_mode"`
		SubscribersMode      bool      `json:"subscribers_mode"`
		UpdatedAt            time.Time `json:"updated_at"`
	} `json:"chatroom"`
	FollowerBadges     []any `json:"follower_badges"`
	FollowersCount     int   `json:"followers_count"`
	ID                 int   `json:"id"`
	IsAffiliate        bool  `json:"is_affiliate"`
	IsBanned           bool  `json:"is_banned"`
	Livestream         any   `json:"livestream"`
	Muted              bool  `json:"muted"`
	OfflineBannerImage struct {
		Src    string `json:"src"`
		Srcset string `json:"srcset"`
	} `json:"offline_banner_image"`
	PlaybackURL      string `json:"playback_url"`
	RecentCategories []struct {
		Banner struct {
			Responsive string `json:"responsive"`
			URL        string `json:"url"`
		} `json:"banner"`
		Category struct {
			Icon string `json:"icon"`
			ID   int    `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"category"`
		CategoryID  int      `json:"category_id"`
		DeletedAt   any      `json:"deleted_at"`
		Description any      `json:"description"`
		ID          int      `json:"id"`
		IsMature    bool     `json:"is_mature"`
		IsPromoted  bool     `json:"is_promoted"`
		Name        string   `json:"name"`
		Slug        string   `json:"slug"`
		Tags        []string `json:"tags"`
		Viewers     int      `json:"viewers"`
	} `json:"recent_categories"`
	Role             any    `json:"role"`
	Slug             string `json:"slug"`
	SubscriberBadges []struct {
		BadgeImage struct {
			Src    string `json:"src"`
			Srcset string `json:"srcset"`
		} `json:"badge_image"`
		ChannelID int `json:"channel_id"`
		ID        int `json:"id"`
		Months    int `json:"months"`
	} `json:"subscriber_badges"`
	SubscriptionEnabled bool `json:"subscription_enabled"`
	User                struct {
		AgreedToTerms   bool      `json:"agreed_to_terms"`
		Bio             string    `json:"bio"`
		City            any       `json:"city"`
		Country         any       `json:"country"`
		Discord         string    `json:"discord"`
		EmailVerifiedAt time.Time `json:"email_verified_at"`
		Facebook        string    `json:"facebook"`
		ID              int       `json:"id"`
		Instagram       string    `json:"instagram"`
		ProfilePic      string    `json:"profile_pic"`
		State           any       `json:"state"`
		Tiktok          string    `json:"tiktok"`
		Twitter         string    `json:"twitter"`
		Username        string    `json:"username"`
		Youtube         string    `json:"youtube"`
	} `json:"user"`
	UserID     int  `json:"user_id"`
	Verified   bool `json:"verified"`
	VodEnabled bool `json:"vod_enabled"`
	CustomerID string
	ContentID  string
}

func (c *Client) Channel(channel string) (*Channel, error) {
	url := fmt.Sprintf("https://kick.com/api/v2/channels/%s", channel)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data Channel
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.PlaybackURL != "" {
		data.ContentID, data.CustomerID = getChannelPlaybackSignatures(data.PlaybackURL)
	}

	return &data, nil
}

func getChannelPlaybackSignatures(playbackURL string) (string, string) {
	parsed, _ := url.Parse(playbackURL)
	parts := strings.Split(parsed.Path, ".")
	if len(parts) > 4 {
		return parts[1], parts[3]
	}
	return "", ""
}
