package downloader

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/Kostaaa1/twitch/pkg/spinner"
)

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
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

	playlist.TruncateSegments(mu.Start, mu.End)

	jobsChan := make(chan segmentJob)
	resultsChan := make(chan segmentJob)

	go func() {
		for i, seg := range playlist.Segments {
			if strings.HasSuffix(seg.URL, ".ts") {
				lastIndex := strings.LastIndex(variant.URL, "/")
				fullSegURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], seg.URL)
				select {
				case <-dl.ctx.Done():
					return
				case jobsChan <- segmentJob{
					index: i,
					url:   fullSegURL,
				}:
				}
			}
		}
		close(jobsChan)
	}()

	const maxWorkers = 8
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-dl.ctx.Done():
					return
				case job, ok := <-jobsChan:
					if !ok {
						return
					}
					status, data, err := dl.fetchWithStatus(job.url)
					if status == http.StatusForbidden {
						switch {
						case strings.Contains(job.url, "unmuted"):
							job.url = strings.Replace(job.url, "-unmuted", "", 1)
							data, err = dl.fetch(job.url)
						case strings.Contains(job.url, "muted"):
							job.url = strings.Replace(job.url, "-muted", "", 1)
							data, err = dl.fetch(job.url)
						}
					}

					job.err = err
					job.data = data

					select {
					case <-dl.ctx.Done():
						return
					case resultsChan <- job:
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	segmentBuffer := make(map[int]segmentJob)
	nextIndexToWrite := 0

	for {
		select {
		case <-dl.ctx.Done():
			return nil
		case result, ok := <-resultsChan:
			if !ok {
				return nil
			}
			if result.err != nil {
				return fmt.Errorf("error downloading segment %s: %v", result.url, result.err)
			}

			segmentBuffer[result.index] = result

			for {
				if job, exists := segmentBuffer[nextIndexToWrite]; exists {
					n, err := mu.Writer.Write(job.data)
					if err != nil {
						return fmt.Errorf("error writing segment %d: %v", nextIndexToWrite, err)
					}

					delete(segmentBuffer, nextIndexToWrite)
					nextIndexToWrite++

					msg := spinner.ChannelMessage{Bytes: int64(n)}
					mu.NotifyProgressChannel(msg, dl.progressCh)
				} else {
					break
				}
			}
		}
	}
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

	playlist.TruncateSegments(mu.Start, mu.End)

	for _, seg := range playlist.Segments {
		if strings.HasSuffix(seg.URL, ".ts") {
			lastIndex := strings.LastIndex(variant.URL, "/")
			fullSegURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], seg.URL)
			n, err := dl.download(fullSegURL, mu.Writer)
			if err != nil {
				log.Fatal(err)
				return err
			}
			msg := spinner.ChannelMessage{Bytes: n}
			mu.NotifyProgressChannel(msg, dl.progressCh)
		}
	}

	return nil
}
