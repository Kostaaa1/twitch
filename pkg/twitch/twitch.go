package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/Kostaaa1/twitch/internal/config"
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
	progressCh  chan ProgresbarChanData
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

func (tw *API) GetToken() string {
	return fmt.Sprintf("Bearer %s", tw.config.User.Creds.AccessToken)
}

func (tw *API) BatchDownload(units []MediaUnit) {
	climit := runtime.NumCPU() / 2
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

	switch unit.Type {
	case TypeVOD:
		err = tw.ParallelVodDownload(unit)
	case TypeClip:
		err = tw.DownloadClip(unit)
	case TypeLivestream:
		err = tw.RecordStream(unit)
	}

	if err != nil {
		logFile, e := os.OpenFile("errors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if e != nil {
			log.Fatal("Error opening log file: ", e)
		}
		defer logFile.Close()

		logger := log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		msg := fmt.Sprintf("Failed to download - %s. ERROR: %v", unit.Slug, err)
		logger.Printf(msg, unit.Slug, err)
	}

	if file, ok := unit.W.(*os.File); ok && file != nil {
		tw.progressCh <- ProgresbarChanData{
			Text:   file.Name(),
			Error:  err,
			IsDone: true,
		}
	}
}

func (api *API) downloadAndWriteSegment(segmentURL string, w io.Writer) (int64, error) {
	resp, err := api.client.Get(segmentURL)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	return io.Copy(w, resp.Body)
}

func (api *API) downloadSegmentToTempFile(segment, vodPlaylistURL, tempDir string, unit MediaUnit) error {
	lastIndex := strings.LastIndex(vodPlaylistURL, "/")
	segmentURL := fmt.Sprintf("%s/%s", vodPlaylistURL[:lastIndex], segment)
	tempFilePath := fmt.Sprintf("%s/%s", tempDir, segmentFileName(segment))

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file %s: %w", tempFilePath, err)
	}
	defer tempFile.Close()

	n, err := api.downloadAndWriteSegment(segmentURL, tempFile)
	if err != nil {
		return fmt.Errorf("error downloading segment %s: %w", segmentURL, err)
	}

	if f, ok := unit.W.(*os.File); ok && f != nil {
		api.progressCh <- ProgresbarChanData{
			Text:  f.Name(),
			Bytes: n,
		}
	}

	return nil
}
