package kick

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	utls "github.com/refraction-networking/utls"
)

type Client struct {
	client *http.Client
	progCh chan spinner.ChannelMessage
	ctx    context.Context
}

func (c *Client) SetProgressChannel(progCh chan spinner.ChannelMessage) {
	c.progCh = progCh
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
	StartTime    Datetime    `json:"start_time"`
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

type Datetime struct {
	time.Time
}

func (d *Datetime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "close")
}
