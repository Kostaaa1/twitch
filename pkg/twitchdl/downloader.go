package twitchdl

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Downloader struct {
	TWApi      *twitch.Client
	progressCh chan spinner.ChannelMessage
	httpClient *http.Client
	config     Config
}

type Config struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
	SpinnerModel    string `json:"spinner_model"`
	SkipAds         bool   `json:"skip_ads"`
}

// func (dl *Downloader) SetConfig(conf config.Downloader) {
// 	dl.config = conf
// }

func New(twClient *twitch.Client, httpClient *http.Client, conf Config) *Downloader {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Downloader{
		TWApi:      twClient,
		httpClient: httpClient,
		config:     conf,
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

func (mu *Unit) recordStream(dl *Downloader) error {
	isLive, err := dl.TWApi.IsChannelLive(mu.ID)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", mu.ID)
	}

	master, err := dl.TWApi.MasterPlaylistStream(mu.ID)
	if err != nil {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(mu.Quality.String())
	if err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	count := 0
	maxCount := 1
	var byteBuf bytes.Buffer

	for range ticker.C {
		b, err := dl.fetch(variant.URL)
		if err != nil {
			msg := spinner.ChannelMessage{Error: errors.New("stream ended")}
			mu.NotifyProgressChannel(msg, dl.progressCh)
			return nil
		}

		segments := strings.Split(string(b), "\n")
		lastSegInfo := strings.TrimPrefix(segments[len(segments)-3], "#EXTINF:")

		if dl.config.SkipAds && strings.Contains(lastSegInfo, "Amazon") {
			msg := spinner.ChannelMessage{Message: "[Ad is running]", Bytes: 0}
			mu.NotifyProgressChannel(msg, dl.progressCh)
			continue
		}

		parts := strings.SplitN(lastSegInfo, ",", 2)
		val, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		maxCount = int(val)

		if maxCount <= 0 {
			maxCount = 1
		}

		if count == 0 {
			tsURL := segments[len(segments)-2]
			segmentBytes, _ := dl.fetch(tsURL)
			byteBuf.Reset()
			byteBuf.Write(segmentBytes)
		}

		segmentSize := byteBuf.Len() / maxCount
		start := count * segmentSize
		end := start + segmentSize
		if end > byteBuf.Len() {
			end = byteBuf.Len()
		}

		n, err := mu.Writer.Write(byteBuf.Bytes()[start:end])
		if err != nil {
			return err
		}

		msg := spinner.ChannelMessage{Bytes: int64(n)}
		mu.NotifyProgressChannel(msg, dl.progressCh)

		count++
		if count == maxCount {
			count = 0
		}
	}

	return nil
}

func (dl *Downloader) fetch(u string) ([]byte, error) {
	resp, err := dl.httpClient.Get(u)
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
	resp, err := dl.httpClient.Get(u)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}
	return io.Copy(w, resp.Body)
}
