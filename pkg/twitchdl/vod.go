package twitchdl

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/Kostaaa1/twitch/internal/m3u8"
	"github.com/Kostaaa1/twitch/internal/spinner"
)

func segmentFileName(segmentURL string) string {
	parts := strings.Split(segmentURL, "/")
	return parts[len(parts)-1]
}

func (mu MediaUnit) parallelVodDownload(dl *Downloader) error {
	if mu.ID == "" {
		return errors.New("slug is required for vod media list")
	}

	master, status, err := dl.api.GetVODMasterM3u8(mu.ID)
	if err != nil && status != http.StatusForbidden {
		return err
	}

	variant, err := master.GetVariantPlaylistByQuality(mu.Quality)
	if err != nil {
		return err
	}

	mp, err := dl.fetch(variant.URL)
	if err != nil {
		return err
	}

	media := m3u8.ParseMediaPlaylist(string(mp))
	if err := media.TruncateSegments(mu.Start, mu.End); err != nil {
		return err
	}

	tempDir, _ := os.MkdirTemp("", fmt.Sprintf("vod_segments_%s", mu.ID))
	defer os.RemoveAll(tempDir)

	maxConcurrency := runtime.NumCPU() / 2
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, segURL := range media.Segments {
		if strings.HasSuffix(segURL, ".ts") {
			wg.Add(1)

			go func(segURL string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				if err := dl.downloadSegmentToTempFile(segURL, variant.URL, tempDir, mu); err != nil {
					fmt.Println(err)
				}
			}(segURL)
		}
	}

	wg.Wait()

	if err := mu.writeSegmentsToOutput(media.Segments, tempDir); err != nil {
		return err
	}

	return nil
}

func (mu MediaUnit) writeSegmentsToOutput(segments []string, tempDir string) error {
	for _, segURL := range segments {
		if !strings.HasSuffix(segURL, ".ts") {
			continue
		}
		tempFilePath := fmt.Sprintf("%s/%s", tempDir, segmentFileName(segURL))
		tempFile, err := os.Open(tempFilePath)
		if err != nil {
			return fmt.Errorf("failed to open temp file %s: %w", tempFilePath, err)
		}
		if _, err := io.Copy(mu.Writer, tempFile); err != nil {
			tempFile.Close()
			return fmt.Errorf("failed to write segment to output file: %w", err)
		}
		tempFile.Close()
	}
	return nil
}

func (mu MediaUnit) StreamVOD(dl *Downloader) error {
	if mu.ID == "" {
		return errors.New("slug is required for vod media list")
	}

	master, status, err := dl.api.GetVODMasterM3u8(mu.ID)
	if err != nil && status != http.StatusForbidden {
		return err
	}

	variant, err := master.GetVariantPlaylistByQuality(mu.Quality)
	if err != nil {
		return err
	}

	mediaPlaylist, err := dl.fetch(variant.URL)
	if err != nil {
		return err
	}

	playlist := m3u8.ParseMediaPlaylist(string(mediaPlaylist))
	if err := playlist.TruncateSegments(mu.Start, mu.End); err != nil {
		return err
	}

	for _, segment := range playlist.Segments {
		if strings.HasSuffix(segment, ".ts") {
			lastIndex := strings.LastIndex(variant.URL, "/")
			segmentURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], segment)

			n, err := dl.downloadAndWriteSegment(segmentURL, mu.Writer)
			if err != nil {
				fmt.Printf("error downloading segment %s: %v\n", segmentURL, err)
				return err
			}

			if file, ok := mu.Writer.(*os.File); ok && file != nil {
				dl.progressCh <- spinner.ChannelMessage{
					Text:  file.Name(),
					Bytes: n,
				}
			}
		}
	}

	return nil
}
