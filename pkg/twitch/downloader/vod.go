package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/Kostaaa1/twitch/pkg/m3u8"
	"golang.org/x/sync/errgroup"
)

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

func (dl *Downloader) getVariantAndMediaPlaylistForUnit(unit Unit) (variant *m3u8.VariantPlaylist, media *m3u8.MediaPlaylist, err error) {
	master, err := dl.twClient.MasterPlaylistVOD(unit.ID)
	if err != nil {
		return nil, nil, err
	}

	variant, err = master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return nil, nil, err
	}

	media, err = dl.twClient.FetchAndParseMediaPlaylist(variant)
	if err != nil {
		return nil, nil, err
	}
	media.TruncateSegments(unit.Start, unit.End)

	return variant, media, nil
}

func buildTSURL(baseURL, segment string) string {
	lastIndex := strings.LastIndex(baseURL, "/")
	return fmt.Sprintf("%s/%s", baseURL[:lastIndex], segment)
}

func (dl *Downloader) fetchSegmentWithRetry(ctx context.Context, u string) (io.ReadCloser, error) {
	status, data, err := dl.fetchWithStatusCloser(ctx, u)
	if status == http.StatusForbidden {
		switch {
		case strings.Contains(u, "unmuted"):
			u = strings.Replace(u, "-unmuted", "-muted", 1)
			_, data, err = dl.fetchWithStatusCloser(ctx, u)
		case strings.Contains(u, "muted"):
			u = strings.Replace(u, "-muted", "", 1)
			_, data, err = dl.fetchWithStatusCloser(ctx, u)
		}
	}

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (dl *Downloader) downloadVOD(ctx context.Context, unit Unit) error {
	variant, playlist, _ := dl.getVariantAndMediaPlaylistForUnit(unit)

	ctx, cancel := context.WithCancel(ctx)
	abort := func(err error) error {
		fmt.Println("ABORT CALLEDJ", err)
		cancel()
		return err
	}

	currentChunk := atomic.Uint32{}
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < 8; i++ {
		g.Go(func() error {
			for {
				chunkInx := int(currentChunk.Add(1) - 1)
				if chunkInx >= len(playlist.Segments) {
					abort(nil)
					return nil
				}

				seg := playlist.Segments[chunkInx]

				if strings.HasSuffix(seg.URL, ".ts") {
					fullSegURL := buildTSURL(variant.URL, seg.URL)

					r, err := dl.fetchSegmentWithRetry(ctx, fullSegURL)
					if err != nil {
						abort(err)
						return err
					}

					fmt.Println("Sending to channel: ", r)
					seg.Data <- r
					close(seg.Data)
				}
			}
		})
	}

	g.Go(func() error {
		for i := 0; i < len(playlist.Segments); i++ {
			select {
			case <-ctx.Done():
				return nil
			case chunk, ok := <-playlist.Segments[i].Data:
				if !ok {
					continue
				}

				n, err := io.Copy(unit.Writer, chunk)
				chunk.Close()
				if err != nil {
					abort(err)
					return err
				}

				fmt.Printf("Written: %d\n", n)
			}
		}

		return nil
	})

	return g.Wait()
}

// TODO: batch writes / buffered writer / temp memory-mapped file / sliding windows writer (?)
// func (dl *Downloader) downloadVOD(ctx context.Context, unit Unit) error {
// 	variant, playlist, err := dl.getVariantAndMediaPlaylistForUnit(unit)
// 	if err != nil {
// 		return err
// 	}

// 	jobsChan := make(chan segmentJob)
// 	resultsChan := make(chan segmentJob)

// 	go func() {
// 		for i, seg := range playlist.Segments {
// 			if strings.HasSuffix(seg.URL, ".ts") {
// 				lastIndex := strings.LastIndex(variant.URL, "/")
// 				segURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], seg.URL)

// 				select {
// 				case <-ctx.Done():
// 					return
// 				case jobsChan <- segmentJob{
// 					index: i,
// 					url:   segURL,
// 				}:
// 				}
// 			}
// 		}
// 		close(jobsChan)
// 	}()

// 	const maxWorkers = 8
// 	var wg sync.WaitGroup

// 	for i := 0; i < maxWorkers; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			for {
// 				select {
// 				case <-ctx.Done():
// 					return
// 				case job, ok := <-jobsChan:
// 					if !ok {
// 						return
// 					}

// 					// TODO: NOT TESTED.. 403 when fetching segments that have unmuted or muted...
// 					status, data, err := dl.fetchWithStatus(ctx, job.url)
// 					if status == http.StatusForbidden {
// 						switch {
// 						case strings.Contains(job.url, "unmuted"):
// 							job.url = strings.Replace(job.url, "-unmuted", "-muted", 1)
// 							data, err = dl.fetch(ctx, job.url)
// 						case strings.Contains(job.url, "muted"):
// 							job.url = strings.Replace(job.url, "-muted", "", 1)
// 							data, err = dl.fetch(ctx, job.url)
// 						}
// 					}

// 					job.err = err
// 					job.data = data

// 					select {
// 					case <-ctx.Done():
// 						return
// 					case resultsChan <- job:
// 					}
// 				}
// 			}
// 		}()
// 	}

// 	go func() {
// 		wg.Wait()
// 		close(resultsChan)
// 	}()

// 	segmentBuffer := make(map[int]segmentJob)
// 	nextIndexToWrite := 0

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return nil
// 		case result, ok := <-resultsChan:
// 			if !ok {
// 				return nil
// 			}
// 			if result.err != nil {
// 				return fmt.Errorf("error downloading segment %s: %v", result.url, result.err)
// 			}

// 			segmentBuffer[result.index] = result

// 			for {
// 				job, exists := segmentBuffer[nextIndexToWrite]
// 				if !exists {
// 					break
// 				}
// 				delete(segmentBuffer, nextIndexToWrite)
// 				nextIndexToWrite++

// 				errCh := make(chan error, 1)
// 				go func(data []byte) {
// 					_, err := unit.Writer.Write(job.data)
// 					errCh <- err
// 				}(job.data)

// 				select {
// 				case <-ctx.Done():
// 					return ctx.Err()
// 				case err := <-errCh:
// 					if err != nil {
// 						return err
// 					}
// 				}

// 				// msg := spinner.Message{
// 				// 	ID:    unit.GetTitle(),
// 				// 	Bytes: int64(len(job.data)),
// 				// }
// 				// dl.NotifyProgressChannel(msg, unit)
// 			}
// 		}
// 	}
// }

func (unit Unit) StreamVOD(ctx context.Context, dl *Downloader) error {
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

			// resp, err := dl.twClient.HttpClient().Get(fullSegURL)
			// if err != nil {
			// 	return err
			// }
			// defer resp.Body.Close()

			b, err := dl.fetch(ctx, fullSegURL)
			if err != nil {
				return err
			}

			_, err = io.Copy(unit.Writer, bytes.NewReader(b))
			if err != nil {
				return err
			}

			// msg := spinner.Message{ID: unit.GetID(), Bytes: n}
			// dl.NotifyProgressChannel(msg, unit)
		}
	}

	return nil
}
