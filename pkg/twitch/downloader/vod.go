package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/Kostaaa1/twitch/pkg/m3u8"
	"golang.org/x/sync/errgroup"
)

func (dl *Downloader) getPlaylistsForUnit(ctx context.Context, unit Unit) (variant *m3u8.VariantPlaylist, media *m3u8.MediaPlaylist, err error) {
	master, err := dl.twClient.MasterPlaylistVOD(ctx, unit.ID)
	if err != nil {
		return nil, nil, err
	}

	variant, err = master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return nil, nil, err
	}

	media, err = dl.twClient.FetchAndParseMediaPlaylist(variant.URL)
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
	data, status, err := dl.fetch(ctx, u)
	if status == http.StatusForbidden {
		switch {
		case strings.Contains(u, "unmuted"):
			u = strings.Replace(u, "-unmuted", "-muted", 1)
			data, _, err = dl.fetch(ctx, u)
		case strings.Contains(u, "muted"):
			u = strings.Replace(u, "-muted", "", 1)
			data, _, err = dl.fetch(ctx, u)
		}
	}

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (dl *Downloader) downloadVOD(ctx context.Context, unit Unit) error {
	variant, playlist, _ := dl.getPlaylistsForUnit(ctx, unit)

	g, ctx := errgroup.WithContext(ctx)
	currentChunk := atomic.Uint32{}

	for i := 0; i < 8; i++ {
		g.Go(func() error {
			for {
				chunkInx := int(currentChunk.Add(1) - 1)
				if chunkInx >= len(playlist.Segments) {
					return nil
				}

				seg := playlist.Segments[chunkInx]

				if strings.HasSuffix(seg.URL, ".ts") {
					fullSegURL := buildTSURL(variant.URL, seg.URL)

					reader, err := dl.fetchSegmentWithRetry(ctx, fullSegURL)
					if err != nil {
						return err
					}

					seg.Data <- reader
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

				// chunk.Close()

				n, err := io.Copy(unit.Writer, chunk)
				if err != nil {
					return err
				}

				dl.notify(Progress{
					ID:    unit.GetID(),
					Err:   unit.Error,
					Bytes: n,
				})
			}
		}

		return nil
	})

	return g.Wait()
}

func (unit Unit) StreamVOD(ctx context.Context, dl *Downloader) error {
	master, err := dl.twClient.MasterPlaylistVOD(ctx, unit.ID)
	if err != nil {
		return err
	}

	variant, err := master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return err
	}

	playlist, err := dl.twClient.FetchAndParseMediaPlaylist(variant.URL)
	if err != nil {
		return err
	}
	playlist.TruncateSegments(unit.Start, unit.End)

	for _, seg := range playlist.Segments {
		if strings.HasSuffix(seg.URL, ".ts") {
			lastIndex := strings.LastIndex(variant.URL, "/")
			fullSegURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], seg.URL)

			reader, _, err := dl.fetch(ctx, fullSegURL)
			if err != nil {
				return err
			}

			_, err = io.Copy(unit.Writer, reader)
			if err != nil {
				return err
			}

			// msg := spinner.Message{ID: unit.GetID(), Bytes: n}
			// dl.NotifyProgressChannel(msg, unit)
		}
	}

	return nil
}

// TODO: batch writes / buffered writer / temp memory-mapped file / sliding windows writer (?)
// func (dl *Downloader) downloadVOD(ctx context.Context, unit Unit) error {
// 	variant, playlist, err := dl.getPlaylistsForUnit(unit)
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
