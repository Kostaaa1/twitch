package twitchdl

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Downloader struct {
	TWApi      *twitch.Client
	progressCh chan spinner.ChannelMessage
	config     Config
	done       chan struct{}
}

type Config struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
	SpinnerModel    string `json:"spinner_model"`
	SkipAds         bool   `json:"skip_ads"`
}

func New(done chan struct{}, twClient *twitch.Client, conf Config) *Downloader {
	return &Downloader{
		done:   done,
		TWApi:  twClient,
		config: conf,
	}
}

func (dl *Downloader) SetProgressChannel(progressCh chan spinner.ChannelMessage) {
	dl.progressCh = progressCh
}

func (dl *Downloader) Download(u Unit) error {
	if u.Error == nil {
		switch u.Type {
		case TypeVOD:
			u.Error = dl.downloadVOD(u)
		case TypeClip:
			u.Error = dl.downloadClip(u)
		case TypeLivestream:
			u.Error = dl.recordStream(u)
		}
	}
	return u.Error
}

func (dl *Downloader) Record(unit Unit) error {
	return dl.recordStream(unit)
}

func (dl *Downloader) BatchDownload(units []Unit) {
	climit := runtime.NumCPU() / 2

	var wg sync.WaitGroup
	sem := make(chan struct{}, climit)

	for _, unit := range units {
		wg.Add(1)

		go func(u Unit) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := dl.Download(u); err != nil {
				u.Error = err
			}

			msg := spinner.ChannelMessage{Error: u.Error, IsDone: true}
			u.NotifyProgressChannel(msg, dl.progressCh)
		}(unit)
	}

	wg.Wait()
}

func (dl *Downloader) fetch(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := dl.TWApi.HTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (dl *Downloader) download(url string, w io.Writer) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := dl.TWApi.HTTPClient().Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	return io.Copy(w, resp.Body)
}
