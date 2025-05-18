package downloader

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Kostaaa1/twitch/pkg/spinner"
)

type segmentJob struct {
	index     int
	url       string
	data      []byte
	err       error
	bytesRead int64
}

// Concurrent ordered download. Segments needs to be in order so it can be written to file. Instead of doing this sequentally (one-by-one), this is downloading them concurrently.
func (dl *Downloader) downloadVOD(mu Unit) error {
	master, err := dl.TWApi.MasterPlaylistVOD(mu.ID)
	if err != nil {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(mu.Quality.String())
	if err != nil {
		return err
	}
	playlist, err := dl.TWApi.FetchAndParseMediaPlaylist(variant)
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
					msg := spinner.ChannelMessage{Bytes: job.bytesRead}
					mu.NotifyProgressChannel(msg, dl.progressCh)
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

func (mu Unit) StreamVOD(dl *Downloader) error {
	master, err := dl.TWApi.MasterPlaylistVOD(mu.ID)
	if err != nil {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(mu.Quality.String())
	if err != nil {
		return err
	}
	playlist, err := dl.TWApi.FetchAndParseMediaPlaylist(variant)
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
			n, err := dl.download(segmentURL, mu.Writer)
			if err != nil {
				fmt.Printf("error downloading segment %s: %v\n", segmentURL, err)
				return err
			}
			msg := spinner.ChannelMessage{Bytes: n}
			mu.NotifyProgressChannel(msg, dl.progressCh)
		}
	}

	return nil
}
