package twitchdl

import (
	"fmt"
	"io"
	"sync"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Downloader struct {
	TWApi      *twitch.Client
	progressCh chan spinner.ChannelMessage
	config     Config
}

type Config struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
	SpinnerModel    string `json:"spinner_model"`
	SkipAds         bool   `json:"skip_ads"`
}

func New(twClient *twitch.Client, conf Config) *Downloader {
	return &Downloader{
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
			u.Error = u.downloadVOD(dl)
		case TypeClip:
			u.Error = u.downloadClip(dl)
		case TypeLivestream:
			u.Error = u.recordStream(dl)
		}
	}
	return u.Error
}

func (dl *Downloader) BatchDownload(units []Unit) {
	// climit := runtime.NumCPU() / 2
	climit := 25

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

func (dl *Downloader) fetch(u string) ([]byte, error) {
	resp, err := dl.TWApi.HTTPClient().Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (dl *Downloader) download(u string, w io.Writer) (int64, error) {
	resp, err := dl.TWApi.HTTPClient().Get(u)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}
	return io.Copy(w, resp.Body)
}
