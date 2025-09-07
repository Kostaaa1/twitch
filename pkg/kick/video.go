package kick

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (c *Client) MasterPlaylistURL(input string) (string, error) {
	channel, vodUUID := "", ""

	// if input is UUID then get channel from V1Channel, otherwise if URL parse channel name and UUID from path
	if uuid.Validate(input) == nil {
		// TODO: not working
		data, err := c.V1Channel(input)
		if err != nil {
			return "", err
		}
		vodUUID = input
		channel = data.Livestream.Channel.Slug
	} else {
		parsed, err := url.Parse(input)
		if err != nil {
			return "", err
		}
		parts := strings.Split(parsed.Path, "/")
		channel = parts[1]
		vodUUID = parts[3]
	}

	if channel == "" || vodUUID == "" {
		return "", errors.New("invalid kick URL")
	}

	videos, err := c.Videos(channel)
	if err != nil {
		return "", err
	}

	for _, data := range videos {
		if data.Video.UUID == vodUUID {
			return data.Source, nil
		}
	}

	return "", errors.New("master.m3u8 not found")
}

func (c *Client) Videos(channel string) ([]*VideoMetadata, error) {
	url := fmt.Sprintf("https://kick.com/api/v2/channels/%s/videos", channel)

	var videos []*VideoMetadata
	if err := c.sendRequestAndDecode(url, http.MethodGet, &videos); err != nil {
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

			if channelData != nil && video.Thumbnail.Src != "" {
				vodSig := getVideoSignature(video.Thumbnail.Src)

				// TODO: this does not work properly! The timestamp of StartTime Hour and Minute is sometimes wrong
				minute := video.StartTime.Minute()
				if video.StartTime.Second() < 10 {
					minute--
				}

				video.Source = fmt.Sprintf("https://stream.kick.com/ivs/v1/%s/%s/%d/%d/%d/%d/%d/%s/media/hls/master.m3u8",
					channelData.CustomerID,
					channelData.ContentID,
					video.StartTime.Year(),
					video.StartTime.Month(),
					video.StartTime.Day(),
					video.StartTime.Hour(),
					minute,
					vodSig)
			}
		}
	}

	return videos, nil
}

func getVideoSignature(thumbnailURL string) string {
	parsed, _ := url.Parse(thumbnailURL)
	parts := strings.Split(parsed.Path, "/")
	return parts[len(parts)-2]
}

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

// func (c *Client) VideoByID(id string) (interface{}, error) {
// 	videoURL := fmt.Sprintf("https://kick.com/api/v1/video/%s", id)

// 	req, err := http.NewRequest(http.MethodGet, videoURL, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	setDefaultHeaders(req)

// 	resp, err := c.client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	var video Video
// 	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
// 		return nil, err
// 	}

// 	return video, nil
// }

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
	u := fmt.Sprintf("https://kick.com/api/v2/channels/%s", channel)

	var data Channel
	if err := c.sendRequestAndDecode(u, http.MethodGet, &data); err != nil {
		return nil, err
	}

	if data.PlaybackURL != "" {
		data.CustomerID, data.ContentID = getChannelPlaybackSignatures(data.PlaybackURL)
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

type V1Channel struct {
	ID                int         `json:"id"`
	LiveStreamID      int         `json:"live_stream_id"`
	Slug              interface{} `json:"slug"`
	Thumb             interface{} `json:"thumb"`
	S3                interface{} `json:"s3"`
	TradingPlatformID interface{} `json:"trading_platform_id"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	UUID              string      `json:"uuid"`
	Views             int         `json:"views"`
	DeletedAt         interface{} `json:"deleted_at"`
	IsPruned          bool        `json:"is_pruned"`
	IsPrivate         bool        `json:"is_private"`
	Status            string      `json:"status"`
	Source            string      `json:"source"`
	Livestream        struct {
		ID            int         `json:"id"`
		Slug          string      `json:"slug"`
		ChannelID     int         `json:"channel_id"`
		CreatedAt     string      `json:"created_at"`
		SessionTitle  string      `json:"session_title"`
		IsLive        bool        `json:"is_live"`
		RiskLevelID   interface{} `json:"risk_level_id"`
		StartTime     time.Time   `json:"start_time"`
		Source        interface{} `json:"source"`
		TwitchChannel interface{} `json:"twitch_channel"`
		Duration      int         `json:"duration"`
		Language      string      `json:"language"`
		IsMature      bool        `json:"is_mature"`
		ViewerCount   int         `json:"viewer_count"`
		Tags          []string    `json:"tags"`
		Thumbnail     string      `json:"thumbnail"`
		Channel       struct {
			ID                  int         `json:"id"`
			UserID              int         `json:"user_id"`
			Slug                string      `json:"slug"`
			IsBanned            bool        `json:"is_banned"`
			PlaybackURL         string      `json:"playback_url"`
			NameUpdatedAt       interface{} `json:"name_updated_at"`
			VodEnabled          bool        `json:"vod_enabled"`
			SubscriptionEnabled bool        `json:"subscription_enabled"`
			IsAffiliate         bool        `json:"is_affiliate"`
			FollowersCount      int         `json:"followersCount"`
			User                struct {
				Profilepic string `json:"profilepic"`
				Bio        string `json:"bio"`
				Twitter    string `json:"twitter"`
				Facebook   string `json:"facebook"`
				Instagram  string `json:"instagram"`
				Youtube    string `json:"youtube"`
				Discord    string `json:"discord"`
				Tiktok     string `json:"tiktok"`
				Username   string `json:"username"`
			} `json:"user"`
			CanHost  bool `json:"can_host"`
			Verified struct {
				ID        int       `json:"id"`
				ChannelID int       `json:"channel_id"`
				CreatedAt time.Time `json:"created_at"`
				UpdatedAt time.Time `json:"updated_at"`
			} `json:"verified"`
		} `json:"channel"`
		Categories []struct {
			ID          int         `json:"id"`
			CategoryID  int         `json:"category_id"`
			Name        string      `json:"name"`
			Slug        string      `json:"slug"`
			Tags        []string    `json:"tags"`
			Description interface{} `json:"description"`
			DeletedAt   interface{} `json:"deleted_at"`
			IsMature    bool        `json:"is_mature"`
			IsPromoted  bool        `json:"is_promoted"`
			Viewers     int         `json:"viewers"`
			Category    struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Slug string `json:"slug"`
				Icon string `json:"icon"`
			} `json:"category"`
		} `json:"categories"`
	} `json:"livestream"`
}

func (c *Client) V1Channel(vodUUID string) (*V1Channel, error) {
	u := fmt.Sprintf("https://kick.com/api/v1/video/%s", vodUUID)

	var data V1Channel
	if err := c.sendRequestAndDecode(u, http.MethodGet, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
