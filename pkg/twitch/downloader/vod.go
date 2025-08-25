package downloader

import (
	"fmt"
	"io"
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

// TODO: batch writes / buffered writer / temp memory-mapped file / sliding windows writer (?)
func (dl *Downloader) downloadVOD(unit Unit) error {
	master, err := dl.twClient.MasterPlaylistVOD(unit.ID)
	if err != nil {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return err
	}
	playlist, err := dl.twClient.FetchAndParseMediaPlaylist(variant)
	if err != nil {
		return err
	}
	playlist.TruncateSegments(unit.Start, unit.End)

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

					// TODO: test this.. 403 when fetching segments that have unmuted or muted...
					status, data, err := dl.fetchWithStatus(job.url)
					if status == http.StatusForbidden {
						switch {
						case strings.Contains(job.url, "unmuted"):
							job.url = strings.Replace(job.url, "-unmuted", "-muted", 1)
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
					delete(segmentBuffer, nextIndexToWrite)
					nextIndexToWrite++

					n, err := unit.Writer.Write(job.data)
					if err != nil {
						return fmt.Errorf("error writing segment: %v", err)
					}

					msg := spinner.ChannelMessage{Bytes: int64(n)}
					unit.NotifyProgressChannel(msg, dl.progressCh)
				} else {
					break
				}
			}
		}
	}
}

func (unit Unit) StreamVOD(dl *Downloader) error {
	master, err := dl.twClient.MasterPlaylistVOD(unit.ID)
	if err != nil {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return err
	}
	playlist, err := dl.twClient.FetchAndParseMediaPlaylist(variant)
	if err != nil {
		return err
	}
	playlist.TruncateSegments(unit.Start, unit.End)

	for _, seg := range playlist.Segments {
		if strings.HasSuffix(seg.URL, ".ts") {
			lastIndex := strings.LastIndex(variant.URL, "/")
			fullSegURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], seg.URL)

			resp, err := dl.twClient.HttpClient().Get(fullSegURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			n, err := io.Copy(unit.Writer, resp.Body)
			if err != nil {
				return err
			}

			msg := spinner.ChannelMessage{Bytes: n}
			unit.NotifyProgressChannel(msg, dl.progressCh)
		}
	}

	return nil
}
