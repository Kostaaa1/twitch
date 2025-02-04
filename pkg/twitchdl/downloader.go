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
	api        *twitch.API
	client     *http.Client
	progressCh chan spinner.ChannelMessage
}

func New() *Downloader {
	return &Downloader{
		api:    twitch.New(),
		client: http.DefaultClient,
	}
}

func (dl *Downloader) SetProgressChannel(progressCh chan spinner.ChannelMessage) {
	dl.progressCh = progressCh
}

func (dl *Downloader) Download(u MediaUnit) error {
	if u.Error == nil {
		switch u.Type {
		case TypeVOD:
			u.Error = u.StreamVOD(dl)
		case TypeClip:
			u.Error = u.downloadClip(dl)
		case TypeLivestream:
			u.Error = u.recordStream(dl)
		}
	}
	return u.Error
}

func (dl *Downloader) BatchDownload(units []MediaUnit) {
	climit := runtime.NumCPU() / 2

	var wg sync.WaitGroup
	sem := make(chan struct{}, climit)

	for _, u := range units {
		wg.Add(1)

		go func(u MediaUnit) {
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

func (mu *MediaUnit) recordStream(dl *Downloader) error {
	isLive, err := dl.api.IsChannelLive(mu.ID)
	if err != nil {
		return err
	}

	if !isLive {
		return fmt.Errorf("%s is offline", mu.ID)
	}

	master, err := dl.api.GetStreamMasterPlaylist(mu.ID)
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

	for {
		select {
		case <-ticker.C:
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
	}
}

// func (dl *Downloader) downloadSegmentToTempFile(segment, vodPlaylistURL, tempDir string, mu MediaUnit) error {
// 	lastIndex := strings.LastIndex(vodPlaylistURL, "/")
// 	segmentURL := fmt.Sprintf("%s/%s", vodPlaylistURL[:lastIndex], segment)
// 	tempFilePath := fmt.Sprintf("%s/%s", tempDir, segmentFileName(segment))
// 	tempFile, err := os.Create(tempFilePath)
// 	if err != nil {
// 		return fmt.Errorf("failed to create temp file %s: %w", tempFilePath, err)
// 	}
// 	defer tempFile.Close()
// 	n, err := dl.downloadAndWriteSegment(segmentURL, tempFile)
// 	if err != nil {
// 		return fmt.Errorf("error downloading segment %s: %w", segmentURL, err)
// 	}
// 	if f, ok := mu.Writer.(*os.File); ok && f != nil {
// 		dl.progressCh <- spinner.ChannelMessage{
// 			Text:  f.Name(),
// 			Bytes: n,
// 		}
// 	}
// 	return nil
// }

func (dl *Downloader) downloadAndWriteSegment(segmentURL string, w io.Writer) (int64, error) {
	resp, err := dl.client.Get(segmentURL)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response: %s", resp.Status)
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

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}
	return bytes, nil
}
