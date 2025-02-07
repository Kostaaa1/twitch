package twitchdl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/internal/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Downloader struct {
	TWApi      *twitch.TWClient
	progressCh chan spinner.ChannelMessage
	client     *http.Client
}

func New() *Downloader {
	return &Downloader{
		TWApi:  twitch.New(),
		client: http.DefaultClient,
	}
}

func (dl *Downloader) SetProgressChannel(progressCh chan spinner.ChannelMessage) {
	dl.progressCh = progressCh
}

func (dl *Downloader) Download(u DownloadUnit) error {
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

func (dl *Downloader) BatchDownload(units []DownloadUnit) {
	climit := runtime.NumCPU() / 2

	var wg sync.WaitGroup
	sem := make(chan struct{}, climit)

	for _, u := range units {
		wg.Add(1)

		go func(u DownloadUnit) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := dl.Download(u); err != nil {
				u.Error = err
			}

			if file, ok := u.Writer.(*os.File); ok && file != nil {
				if u.Error != nil {
					os.Remove(file.Name())
				}
				dl.progressCh <- spinner.ChannelMessage{
					Text:   file.Name(),
					Error:  u.Error,
					IsDone: true,
				}
			}
		}(u)
	}
	wg.Wait()
}

func (mu *DownloadUnit) recordStream(dl *Downloader) error {
	isLive, err := dl.TWApi.IsChannelLive(mu.ID)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", mu.ID)
	}

	master, err := dl.TWApi.GetStreamMasterPlaylist(mu.ID)
	if err != nil {
		return err
	}
	mediaList, err := master.GetVariantPlaylistByQuality(mu.Quality)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	tickCount := 0
	var halfBytes *bytes.Reader

	for range ticker.C {
		tickCount++
		var n int64

		if tickCount%2 != 0 {
			b, err := dl.fetch(mediaList.URL)
			if err != nil {
				fmt.Println("Stream ended: ", err)
				return nil
			}

			segments := strings.Split(string(b), "\n")
			tsURL := segments[len(segments)-2]

			bodyBytes, _ := dl.fetch(tsURL)

			half := len(bodyBytes) / 2
			halfBytes = bytes.NewReader(bodyBytes[half:])

			n, _ = io.Copy(mu.Writer, bytes.NewReader(bodyBytes[:half]))
		}

		if tickCount%2 == 0 && halfBytes.Len() > 0 {
			n, _ = io.Copy(mu.Writer, halfBytes)
			halfBytes.Reset([]byte{})
		}

		if file, ok := mu.Writer.(*os.File); ok && file != nil {
			dl.progressCh <- spinner.ChannelMessage{
				Text:  file.Name(),
				Bytes: n,
			}
		}
	}

	return nil
}

func (dl *Downloader) downloadFromURL(u string, w io.Writer) (int64, error) {
	resp, err := dl.client.Get(u)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}
	return io.Copy(w, resp.Body)
}

func (dl *Downloader) fetch(url string) ([]byte, error) {
	resp, err := dl.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}
