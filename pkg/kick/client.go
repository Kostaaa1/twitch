package kick

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	utls "github.com/refraction-networking/utls"
)

type Client struct {
	client     *http.Client
	progressCh chan spinner.ChannelMessage
}

func (c *Client) SetProgressChannel(progressCh chan spinner.ChannelMessage) {
	c.progressCh = progressCh
}

func NewClient() *Client {
	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := utls.Dial(network, addr, nil)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	return &Client{client: client}
}

type VideoMetadata struct {
	Categories []struct {
		Banner struct {
			Responsive string `json:"responsive"`
			URL        string `json:"url"`
		} `json:"banner"`
		CategoryID  int         `json:"category_id"`
		DeletedAt   interface{} `json:"deleted_at"`
		Description interface{} `json:"description"`
		ID          int         `json:"id"`
		IsMature    bool        `json:"is_mature"`
		IsPromoted  bool        `json:"is_promoted"`
		Name        string      `json:"name"`
		Slug        string      `json:"slug"`
		Tags        []string    `json:"tags"`
		Viewers     int         `json:"viewers"`
	} `json:"categories"`
	ChannelID    int         `json:"channel_id"`
	CreatedAt    string      `json:"created_at"`
	Duration     int         `json:"duration"`
	ID           int         `json:"id"`
	IsLive       bool        `json:"is_live"`
	IsMature     bool        `json:"is_mature"`
	Language     string      `json:"language"`
	RiskLevelID  interface{} `json:"risk_level_id"`
	SessionTitle string      `json:"session_title"`
	Slug         string      `json:"slug"`
	Source       string      `json:"source"`
	StartTime    string      `json:"start_time"`
	Tags         []string    `json:"tags"`
	Thumbnail    struct {
		Src    string `json:"src"`
		Srcset string `json:"srcset"`
	} `json:"thumbnail"`
	TwitchChannel interface{} `json:"twitch_channel"`
	Video         Video       `json:"video"`
	ViewerCount   int         `json:"viewer_count"`
	Views         int         `json:"views"`
}

func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "close")
}

func (c *Client) fetchWithStatus(ctx context.Context, url string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return http.StatusOK, b, err
}
