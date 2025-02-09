package twitchdl

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Kostaaa1/twitch/internal/spinner"
)

type segmentJob struct {
	index     int
	url       string
	data      []byte
	err       error
	bytesRead int64
}

func (mu DownloadUnit) downloadVOD(dl *Downloader) error {
	master, status, err := dl.TWApi.GetVODMasterM3u8(mu.ID)
	if err != nil && status != http.StatusForbidden {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(mu.Quality)
	if err != nil {
		return err
	}
	playlist, err := dl.TWApi.GetVODMediaPlaylist(variant)
	if err != nil {
		return err
	}
	if err := playlist.TruncateSegments(mu.Start, mu.End); err != nil {
		return err
	}

	jobsChan := make(chan segmentJob)
	resultsChan := make(chan segmentJob)

	const maxWorkers = 8
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				data, err := dl.fetch(job.url)
				job.data = data
				job.err = err

				if err == nil {
					job.bytesRead = int64(len(data))
					if file, ok := mu.Writer.(*os.File); ok && file != nil {
						dl.progressCh <- spinner.ChannelMessage{
							Text:  file.Name(),
							Bytes: job.bytesRead,
						}
					}
				}
				resultsChan <- job
			}
		}()
	}

	go func() {
		for i, segment := range playlist.Segments {
			if strings.HasSuffix(segment, ".ts") {
				lastIndex := strings.LastIndex(variant.URL, "/")
				segmentURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], segment)
				jobsChan <- segmentJob{
					index: i,
					url:   segmentURL,
				}
			}
		}
		close(jobsChan)
	}()

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	segmentBuffer := make(map[int]segmentJob)
	nextIndexToWrite := 0

	for result := range resultsChan {
		if result.err != nil {
			return fmt.Errorf("error downloading segment %s: %v", result.url, result.err)
		}

		segmentBuffer[result.index] = result

		for {
			if job, exists := segmentBuffer[nextIndexToWrite]; exists {
				_, err := mu.Writer.Write(job.data)
				if err != nil {
					return fmt.Errorf("error writing segment %d: %v", nextIndexToWrite, err)
				}
				delete(segmentBuffer, nextIndexToWrite)
				nextIndexToWrite++
			} else {
				break
			}
		}
	}

	return nil
}

func (mu DownloadUnit) StreamVOD(dl *Downloader) error {
	master, status, err := dl.TWApi.GetVODMasterM3u8(mu.ID)
	if err != nil && status != http.StatusForbidden {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(mu.Quality)
	if err != nil {
		return err
	}
	playlist, err := dl.TWApi.GetVODMediaPlaylist(variant)
	if err != nil {
		return err
	}

	if err := playlist.TruncateSegments(mu.Start, mu.End); err != nil {
		return err
	}

	for _, segment := range playlist.Segments {
		if strings.HasSuffix(segment, ".ts") {
			lastIndex := strings.LastIndex(variant.URL, "/")
			segmentURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], segment)

			n, err := dl.downloadFromURL(segmentURL, mu.Writer)
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
