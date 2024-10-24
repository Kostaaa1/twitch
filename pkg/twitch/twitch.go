package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/utils"
)

type ProgresbarChanData struct {
	Text   string
	Bytes  int64
	Error  error
	IsDone bool
}

type Client struct {
	config      config.Data
	client      *http.Client
	gqlURL      string
	helixURL    string
	usherURL    string
	decapiURL   string
	gqlClientID string
	mu          sync.Mutex
	progressCh  chan ProgresbarChanData
}

type VideoType int

const (
	TypeClip VideoType = iota
	TypeVOD
	TypeLivestream
)

type MediaUnit struct {
	Slug     string        `json:"input"`
	Vtype    VideoType     `json:"vtype"`
	Quality  string        `json:"quality"`
	Start    time.Duration `json:"start"`
	End      time.Duration `json:"end"`
	DestPath string        `json:"destPath"`
}

func (c *Client) NewMediaUnit(url, quality, output string, start, end time.Duration) (MediaUnit, error) {
	slug, vtype, err := c.ID(url)
	if err != nil {
		return MediaUnit{}, err
	}

	quality = getResolution(quality, vtype)

	if vtype == TypeVOD {
		if start > 0 && end > 0 && start >= end {
			return MediaUnit{}, fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL (%s)", start, end, url)
		}
	}

	dstPath, err := utils.ConstructPathname(output, slug, quality)
	if err != nil {
		return MediaUnit{}, err
	}

	return MediaUnit{
		Slug:     slug,
		Vtype:    vtype,
		Quality:  quality,
		Start:    start,
		End:      end,
		DestPath: dstPath,
	}, nil
}

func (c *Client) ID(URL string) (string, VideoType, error) {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse the URL: %s", err)
	}

	if !strings.Contains(parsedURL.Hostname(), "twitch.tv") {
		return "", 0, fmt.Errorf("the hostname of the URL does not contain twitch.tv")
	}

	s := strings.Split(parsedURL.Path, "/")

	if strings.Contains(parsedURL.Host, "clips.twitch.tv") || strings.Contains(parsedURL.Path, "/clip/") {
		_, id := path.Split(parsedURL.Path)
		return id, TypeClip, nil
	}

	if strings.Contains(parsedURL.Path, "/videos/") {
		_, id := path.Split(parsedURL.Path)
		return id, TypeVOD, nil
	}

	return s[1], TypeLivestream, nil
}

func New() *Client {
	cfg, err := config.Get()
	if err != nil {
		panic(err)
	}

	return &Client{
		client:      http.DefaultClient,
		config:      *cfg,
		gqlURL:      "https://gql.twitch.tv/gql",
		gqlClientID: "kimne78kx3ncx6brgo4mv6wki5h1ko",
		usherURL:    "https://usher.ttvnw.net",
		helixURL:    "https://api.twitch.tv/helix",
		decapiURL:   "https://decapi.me/twitch/uptime",
		mu:          sync.Mutex{},
		progressCh:  nil,
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %s", err)
	}

	if s := resp.StatusCode; s < 200 || s >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status code %d: %s", s, string(b))
	}

	return resp, nil
}

func (c *Client) fetchWithCode(url string) ([]byte, int, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("fetching failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body failed: %w", err)
	}
	return bytes, resp.StatusCode, nil
}

func (c *Client) fetch(url string) ([]byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}
	return bytes, nil
}

func (c *Client) NewGetRequest(URL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *Client) decodeJSONResponse(resp *http.Response, p interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return err
	}
	return nil
}

func (c *Client) sendGqlLoadAndDecode(body *strings.Reader, v any) error {
	req, err := http.NewRequest(http.MethodPost, c.gqlURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request to get the access token: %s", err)
	}
	req.Header.Set("Client-Id", c.gqlClientID)
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	if err := c.decodeJSONResponse(resp, &v); err != nil {
		return err
	}
	return nil
}

func (c *Client) SetProgressChannel(progressCh chan ProgresbarChanData) {
	c.progressCh = progressCh
}

func (c *Client) IsChannelLive(channelName string) (bool, error) {
	u := fmt.Sprintf("%s/%s", c.decapiURL, channelName)

	resp, err := http.Get(u)
	if err != nil {
		return false, fmt.Errorf("failed getting the response from URL: %s. \nError: %s", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("channel %s does not exist?", channelName)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed reading the response Body. \nError: %s", err)
	}

	if strings.HasPrefix(string(b), "[Error from Twitch API]") {
		return false, fmt.Errorf("unexpected error")
	}
	return !strings.Contains(string(b), "offline"), nil
}

func (c *Client) GetToken() string {
	return fmt.Sprintf("Bearer %s", c.config.Creds.AccessToken)
}

func (c *Client) BatchDownload(units []MediaUnit) {
	climit := runtime.GOMAXPROCS(0)

	var wg sync.WaitGroup
	sem := make(chan struct{}, climit)

	for _, unit := range units {
		wg.Add(1)
		go func(unit MediaUnit) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c.Download(unit)
		}(unit)
	}

	wg.Wait()
}

func (c *Client) Download(unit MediaUnit) {
	var err error

	switch unit.Vtype {
	case TypeVOD:
		err = c.DownloadVOD(unit)

	case TypeClip:
		err = c.DownloadClip(unit)

	case TypeLivestream:
		err = c.RecordStream(unit)
	}

	c.progressCh <- ProgresbarChanData{
		Text:   unit.DestPath,
		Error:  err,
		IsDone: true,
	}
}

func (c *Client) downloadSegment(req *http.Request, f *os.File) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get the response from: %s", req.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response status: %s", resp.Status)
	}

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		fmt.Println("Failed to copy to pw: ", err)
		return err
	}

	c.progressCh <- ProgresbarChanData{
		Text:  f.Name(),
		Bytes: n,
	}

	return nil
}
