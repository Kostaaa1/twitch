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

type API struct {
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
	Slug    string
	Vtype   VideoType
	Quality string
	Start   time.Duration
	End     time.Duration
	File    *os.File
}

func (tw *API) NewMediaUnit(URL, quality, output string, start, end time.Duration) (MediaUnit, error) {
	slug, vtype, err := tw.Slug(URL)
	if err != nil {
		return MediaUnit{}, err
	}

	quality = getResolution(quality, vtype)
	if vtype == TypeVOD {
		if start > 0 && end > 0 && start >= end {
			return MediaUnit{}, fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL (%s)", start, end, URL)
		}
	}

	dstPath, err := utils.ConstructPathname(output, slug, quality)
	if err != nil {
		return MediaUnit{}, err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return MediaUnit{}, err
	}

	return MediaUnit{
		Slug:    slug,
		Vtype:   vtype,
		Quality: quality,
		Start:   start,
		End:     end,
		File:    f,
	}, nil
}

func (tw *API) Slug(URL string) (string, VideoType, error) {
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

func New() *API {
	cfg, err := config.Get()
	if err != nil {
		panic(err)
	}

	return &API{
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

func (tw *API) do(req *http.Request) (*http.Response, error) {
	resp, err := tw.client.Do(req)
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

func (tw *API) fetchWithCode(url string) ([]byte, int, error) {
	resp, err := tw.client.Get(url)
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

func (tw *API) fetch(url string) ([]byte, error) {
	resp, err := tw.client.Get(url)
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

func (tw *API) NewGetRequest(URL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (tw *API) decodeJSONResponse(resp *http.Response, p interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return err
	}
	return nil
}

func (tw *API) sendGqlLoadAndDecode(body *strings.Reader, v any) error {
	req, err := http.NewRequest(http.MethodPost, tw.gqlURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request to get the access token: %s", err)
	}

	req.Header.Set("Client-Id", tw.gqlClientID)

	resp, err := tw.do(req)
	if err != nil {
		return err
	}

	if err := tw.decodeJSONResponse(resp, &v); err != nil {
		return err
	}
	return nil
}

func (tw *API) SetProgressChannel(progressCh chan ProgresbarChanData) {
	tw.progressCh = progressCh
}

func (tw *API) IsChannelLive(channelName string) (bool, error) {
	u := fmt.Sprintf("%s/%s", tw.decapiURL, channelName)

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

func (tw *API) GetToken() string {
	return fmt.Sprintf("Bearer %s", tw.config.Creds.AccessToken)
}

func (tw *API) BatchDownload(units []MediaUnit) {
	climit := runtime.GOMAXPROCS(0)

	var wg sync.WaitGroup
	sem := make(chan struct{}, climit)

	for _, unit := range units {
		wg.Add(1)
		go func(unit MediaUnit) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			tw.Download(unit)
		}(unit)
	}

	wg.Wait()
}

func (tw *API) Download(unit MediaUnit) {
	var err error

	switch unit.Vtype {
	case TypeVOD:
		err = tw.downloadVOD(unit)
	case TypeClip:
		err = tw.downloadClip(unit)
	case TypeLivestream:
		err = tw.RecordStream(unit)
	}

	tw.progressCh <- ProgresbarChanData{
		Text:   unit.File.Name(),
		Error:  err,
		IsDone: true,
	}
}

func (api *API) downloadAndWriteSegment(segmentURL string, w io.Writer) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, segmentURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to write to file: %w", err)
	}

	return n, nil
}

func (api *API) downloadSegmentToFile(segmentURL, tempFilePath string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, segmentURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	return io.Copy(tempFile, resp.Body)
}
