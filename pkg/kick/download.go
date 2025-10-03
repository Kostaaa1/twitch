package kick

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/m3u8"
	"golang.org/x/sync/errgroup"
)

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

func (c *Client) getMediaPlaylist(
	ctx context.Context,
	unit Unit,
) (string, *m3u8.MediaPlaylist, error) {
	masterURL, err := c.MasterPlaylistURL(unit.Channel, unit.UUID.String())
	if err != nil {
		return "", nil, err
	}

	res, err := c.cycletls.Do(masterURL, c.defaultCycleTLSOpts(), http.MethodGet)
	if err != nil {
		return "", nil, err
	}

	master := m3u8.Master(res.BodyBytes)

	list, err := master.GetVariantPlaylistByQuality(unit.Quality)
	if err != nil {
		return "", nil, err
	}

	parts := strings.Split(masterURL, "master.m3u8")
	listParts := strings.Split(list.URL, "/")

	basePath := parts[0] + listParts[0]
	playlistURL := parts[0] + list.URL

	res, err = c.cycletls.Do(playlistURL, c.defaultCycleTLSOpts(), http.MethodGet)
	if err != nil {
		return "", nil, err
	}

	playlist := m3u8.ParseMediaPlaylist(bytes.NewReader(res.BodyBytes))
	playlist.TruncateSegments(unit.Start, unit.End)

	return basePath, &playlist, nil
}

func (c *Client) Download(ctx context.Context, u Unit) error {
	// downloadVod is blocking, when done, notify
	err := c.downloadVO(ctx, u)

	c.notify(ProgressMessage{
		ID:    u.GetID(),
		Bytes: 0,
		Error: err,
		Done:  true,
	})

	return err
}

func (c *Client) downloadVO(ctx context.Context, unit Unit) error {
	u, playlist, err := c.getMediaPlaylist(ctx, unit)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	g.Go(func() error {
		for _, seg := range playlist.Segments {
			g.Go(func() error {
				if strings.HasSuffix(seg.URL, ".ts") {
					segmentURL, _ := url.JoinPath(u, seg.URL)

					res, err := c.cycletls.Do(segmentURL, c.defaultCycleTLSOpts(), http.MethodGet)
					if err != nil {
						return err
					}

					seg.Data <- io.NopCloser(bytes.NewReader(res.BodyBytes))
					close(seg.Data)
				}

				return nil
			})
		}
		return nil
	})

	g.Go(func() error {
		for i := 0; i < len(playlist.Segments); i++ {
			select {
			case <-ctx.Done():
				return nil

			case chunk := <-playlist.Segments[i].Data:
				n, err := io.Copy(unit.W, chunk)
				if err != nil {
					return err
				}

				c.notify(ProgressMessage{
					ID:    unit.GetID(),
					Error: unit.GetError(),
					Bytes: int64(n),
					Done:  false,
				})
				chunk.Close()
			}
		}
		return nil
	})

	g.Wait()

	return nil
}

// func (c *Client) downloadVod(ctx context.Context, unit Unit) error {
// 	// MASTER URL NEEDS TO BE FETCHED AND PARSED SO WE CAN GET PLAYLIST QUALITY
// 	// TODO: WHOLE m3u8 PACKAGE NEEDS TO BE IMPROVED
// 	basePath, playlist, err := c.getMediaPlaylist(ctx, unit)
// 	if err != nil {
// 		return err
// 	}

// 	jobsChan := make(chan segmentJob)
// 	resultsChan := make(chan segmentJob)

// 	go func() {
// 		for i, seg := range playlist.Segments {
// 			if strings.HasSuffix(seg.URL, ".ts") {
// 				fullSegURL, _ := url.JoinPath(basePath, seg.URL)
// 				select {
// 				case <-ctx.Done():
// 					return
// 				case jobsChan <- segmentJob{
// 					index: i,
// 					url:   fullSegURL,
// 				}:
// 				}
// 			}
// 		}
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

// 					req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.url, nil)
// 					if err != nil {
// 						return
// 					}

// 					resp, err := c.httpClient.Do(req)
// 					if err != nil {
// 						job.err = err
// 					} else {
// 						data, err := io.ReadAll(resp.Body)
// 						job.err = err
// 						job.data = data
// 						resp.Body.Close()

// 						select {
// 						case <-ctx.Done():
// 							return
// 						case resultsChan <- job:
// 						}
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
// 				if job, exists := segmentBuffer[nextIndexToWrite]; exists {
// 					delete(segmentBuffer, nextIndexToWrite)
// 					nextIndexToWrite++

// 					n, err := unit.W.Write(job.data)
// 					if err != nil {
// 						return fmt.Errorf("error writing segment: %v", err)
// 					}

// 					if ctx.Err() != nil {
// 						return ctx.Err()
// 					}

// 					msg := ProgressMessage{
// 						ID:    unit.GetID(),
// 						Error: unit.GetError(),
// 						Bytes: int64(n),
// 						Done:  false,
// 					}
// 					c.notify(msg)
// 				} else {
// 					break
// 				}
// 			}
// 		}
// 	}
// }
