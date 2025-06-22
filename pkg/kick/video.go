package kick

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/internal/m3u8"
	"github.com/google/uuid"
)

type Video struct {
	CreatedAt         time.Time `json:"created_at"`
	DeletedAt         any       `json:"deleted_at"`
	ID                int       `json:"id"`
	IsPrivate         bool      `json:"is_private"`
	IsPruned          bool      `json:"is_pruned"`
	LiveStreamID      int       `json:"live_stream_id"`
	S3                any       `json:"s3"`
	Slug              any       `json:"slug"`
	Status            string    `json:"status"`
	Thumb             any       `json:"thumb"`
	TradingPlatformID any       `json:"trading_platform_id"`
	UpdatedAt         time.Time `json:"updated_at"`
	UUID              string    `json:"uuid"`
	Views             int       `json:"views"`
}

type VideoWithMetadata struct {
	Categories []struct {
		Banner struct {
			Responsive string `json:"responsive"`
			URL        string `json:"url"`
		} `json:"banner"`
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
	} `json:"categories"`
	ChannelID    int    `json:"channel_id"`
	CreatedAt    string `json:"created_at"`
	Duration     int    `json:"duration"`
	ID           int    `json:"id"`
	IsLive       bool   `json:"is_live"`
	IsMature     bool   `json:"is_mature"`
	Language     string `json:"language"`
	RiskLevelID  any    `json:"risk_level_id"`
	SessionTitle string `json:"session_title"`
	Slug         string `json:"slug"`
	Source       string `json:"source"`
	StartTime    string `json:"start_time"`
	Tags         []any  `json:"tags"`
	Thumbnail    struct {
		Src    string `json:"src"`
		Srcset string `json:"srcset"`
	} `json:"thumbnail"`
	TwitchChannel any   `json:"twitch_channel"`
	Video         Video `json:"video"`
	ViewerCount   int   `json:"viewer_count"`
	Views         int   `json:"views"`
}

func (c *Client) GetVideos(channelName string) ([]VideoWithMetadata, error) {
	url := fmt.Sprintf("https://kick.com/api/v2/channels/%s/videos", channelName)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setDefaultHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var videos []VideoWithMetadata
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return nil, err
	}

	return videos, nil
}

func (c *Client) GetVideoByUUID(id string) (*Video, error) {
	_, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the uuid: %v", err)
	}

	url := fmt.Sprintf("https://kick.com/api/v1/video/%s", id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setDefaultHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var video Video
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		return nil, err
	}

	return &video, nil
}

func (c *Client) getMediaPlaylist(masterURL, quality string) (*m3u8.MediaPlaylist, error) {
	mediaURL, err := buildMediaPlaylistURL(masterURL, quality)
	if err != nil {
		return nil, fmt.Errorf("failed to build media playlist from master m3u8: %v", err)
	}

	resp, err := c.httpClient.Get(mediaURL)
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

func (c *Client) GetMasterPlaylistURL(channel, uuid string) (string, error) {
	videos, err := c.GetVideos(channel)
	if err != nil {
		return "", err
	}

	for _, data := range videos {
		if data.Video.UUID == uuid {
			return data.Source, nil
		}
	}

	return "", errors.New("m3u8 playlist not found")
}

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

type Unit struct {
	Channel string
	VodID   string
	Writer  io.Writer
	Start   time.Duration
	End     time.Duration
}

func (c *Client) downloadVOD(ctx context.Context, unit Unit) error {
	masterURL, err := c.GetMasterPlaylistURL(unit.Channel, unit.VodID)
	if err != nil {
		return err
	}

	playlist, err := c.getMediaPlaylist(masterURL, "1080")
	if err != nil {
		return err
	}

	playlist.TruncateSegments(unit.Start, unit.End)

	jobsChan := make(chan segmentJob)
	resultsChan := make(chan segmentJob)

	go func() {
		for i, seg := range playlist.Segments {
			if strings.HasSuffix(seg.URL, ".ts") {
				// lastIndex := strings.LastIndex(variant.URL, "/")
				// fullSegURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], seg.URL)
				tsURL := strings.TrimSuffix(masterURL, "s")
				select {
				case <-ctx.Done():
					return
				case jobsChan <- segmentJob{
					index: i,
					url:   tsURL,
				}:
				}
			}
		}
		close(jobsChan)
	}()

	const maxWorkers = 8
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobsChan:
					if !ok {
						return
					}

					// status, data, err := dl.fetchWithStatus(job.url)
					// if status == http.StatusForbidden {
					// 	switch {
					// 	case strings.Contains(job.url, "unmuted"):
					// 		job.url = strings.Replace(job.url, "-unmuted", "-muted", 1)
					// 		data, err = dl.fetch(job.url)
					// 	case strings.Contains(job.url, "muted"):
					// 		job.url = strings.Replace(job.url, "-muted", "", 1)
					// 		data, err = dl.fetch(job.url)
					// 	}
					// }

					resp, err := c.httpClient.Get(job.url)
					b, err := io.ReadAll(resp.Body)
					resp.Body.Close()

					job.err = err
					job.data = b

					select {
					case <-ctx.Done():
						return
					case resultsChan <- job:
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	segmentBuffer := make(map[int]segmentJob)
	nextIndexToWrite := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case result, ok := <-resultsChan:
			if !ok {
				return nil
			}
			if result.err != nil {
				return fmt.Errorf("error downloading segment %s: %v", result.url, result.err)
			}

			segmentBuffer[result.index] = result

			for {
				if job, exists := segmentBuffer[nextIndexToWrite]; exists {
					delete(segmentBuffer, nextIndexToWrite)
					nextIndexToWrite++

					_, err := unit.Writer.Write(job.data)
					if err != nil {
						return fmt.Errorf("error writing segment: %v", err)
					}

					// msg := spinner.ChannelMessage{Bytes: int64(n)}
					// mu.NotifyProgressChannel(msg, dl.progressCh)
				} else {
					break
				}
			}
		}
	}
}

func buildMediaPlaylistURL(masterURL, quality string) (string, error) {
	u, err := url.Parse(masterURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse the master URL: %v", err)
	}

	base := strings.TrimSuffix(u.String(), "/master.m3u8")

	switch {
	case strings.HasPrefix("1080", quality):
		return url.JoinPath(base, "1080p", "playlist.m3u8")
	case strings.HasPrefix("720", quality):
		return url.JoinPath(base, "720p", "playlist.m3u8")
	case strings.HasPrefix("480", quality):
		return url.JoinPath(base, "480p", "playlist.m3u8")
	case strings.HasPrefix("360", quality):
		return url.JoinPath(base, "360p", "playlist.m3u8")
	case strings.HasPrefix("160", quality):
		return url.JoinPath(base, "160p", "playlist.m3u8")
	}

	return "", nil
}
