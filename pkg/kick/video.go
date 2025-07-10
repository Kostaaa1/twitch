package kick

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/m3u8"
)

func (c *Client) GetMasterPlaylistURL(channel, vodID string) (string, error) {
	videos, err := c.GetVideos(channel)
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

func (c *Client) GetVideos(channel string) ([]VideoMetadata, error) {
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

	var videos []VideoMetadata
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return nil, err
	}

	return videos, nil
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

func (c *Client) GetVideoByID(id string) (interface{}, error) {
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
