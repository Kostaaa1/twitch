package twitchdl

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
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
	client     *http.Client
}

func New() *Downloader {
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
	}
	return &Downloader{
		TWApi:  twitch.New(),
		client: httpClient,
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

func (mu *Unit) recordStream(dl *Downloader) error {
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

	variant, err := master.GetVariantPlaylistByQuality(mu.Quality.String())
	if err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	count := 0
	maxCount := 1
	writtenBytes := 0

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

		if strings.Contains(lastSegInfo, "Amazon") {
			msg := spinner.ChannelMessage{Message: "Ad is currently running...", Bytes: 0}
			mu.NotifyProgressChannel(msg, dl.progressCh)
			continue
		}

		parts := strings.SplitN(lastSegInfo, ",", 2)
		val, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		maxCount = int(val)

		if count == 0 {
			tsURL := segments[len(segments)-2]
			segmentBytes, _ := dl.fetch(tsURL)
			byteBuf.Reset()
			byteBuf.Write(segmentBytes)
		}

		totalLen := byteBuf.Len()
		portion := totalLen * count / maxCount
		toWrite := portion - writtenBytes

		if toWrite > 0 {
			n, _ := io.Copy(mu.Writer, io.LimitReader(&byteBuf, int64(toWrite)))
			writtenBytes += int(n)
			msg := spinner.ChannelMessage{Bytes: n}
			mu.NotifyProgressChannel(msg, dl.progressCh)
		}
		count++

		if count == maxCount {
			count = 0
			writtenBytes = 0
		}

		// remainder := int64(byteBuf.Len() / (maxCount - count))
		// n, _ := io.Copy(mu.Writer, io.LimitReader(&byteBuf, remainder))
		// count++
		// if count == maxCount {
		// 	count = 0
		// }

		// byteBuf.Reset()
		// byteBuf.Write(segmentBytes)
		// n, _ := io.Copy(mu.Writer, &byteBuf)

		// if segmentDuration == 1 {
		// 	bodyBytes, _ := dl.fetch(tsURL)
		// 	byteBuf.Reset()
		// 	byteBuf.Write(bodyBytes)
		// 	n, _ = io.Copy(mu.Writer, &byteBuf)
		// }

		// if segmentDuration == 2 {
		// 	tickCount++
		// 	if tickCount%2 != 0 {
		// 		bodyBytes, _ := dl.fetch(tsURL)
		// 		half := len(bodyBytes) / 2
		// 		byteBuf.Reset()
		// 		byteBuf.Write(bodyBytes[half:])
		// 		n, _ = io.Copy(mu.Writer, bytes.NewReader(bodyBytes[:half]))
		// 	}
		// 	if tickCount%2 == 0 && byteBuf.Len() > 0 {
		// 		n, _ = io.Copy(mu.Writer, &byteBuf)
		// 		byteBuf.Reset()
		// 	}
		// }
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
