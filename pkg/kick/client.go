package kick

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/Kostaaa1/twitch/pkg/spinner"
)

type Client struct {
	cycletls   cycletls.CycleTLS
	httpClient *http.Client
	ctx        context.Context
	progCh     chan spinner.Message
}

func New() *Client {
	tlsClient := cycletls.Init()
	return &Client{
		cycletls:   tlsClient,
		httpClient: http.DefaultClient,
	}
}

func (c *Client) Close() {
	c.cycletls.Close()
}

func (c *Client) defaultCycleTLSOpts() cycletls.Options {
	return cycletls.Options{
		Ja3:       "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36",
	}
}

func (c *Client) sendRequestAndDecode(URL string, method string, target interface{}) error {
	resp, err := c.cycletls.Do(URL, c.defaultCycleTLSOpts(), method)
	if err != nil {
		return err
	}
	return json.NewDecoder(strings.NewReader(resp.Body)).Decode(target)
}

func (c *Client) SetProgressChannel(progCh chan spinner.Message) {
	c.progCh = progCh
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
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}
