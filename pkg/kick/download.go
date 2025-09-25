package kick

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/Kostaaa1/twitch/pkg/m3u8"
)

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

func (c *Client) fetchWithContext(
	ctx context.Context,
	method string,
	url string,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return b, errors.Join(errors.New("403 forbidden"), err)
	}

	return b, nil
}

func (c *Client) getMediaPlaylist(
	ctx context.Context,
	unit *Unit,
	// channel *string,
	// uuid uuid.UUID,
	// quality string,
) (string, *m3u8.MediaPlaylist, error) {
	masterURL, err := c.MasterPlaylistURL(unit.Channel, unit.UUID.String())
	if err != nil {
		return "", nil, err
	}

	b, err := c.fetchWithContext(ctx, http.MethodGet, masterURL)
	if err != nil {
		return "", nil, err
	}

	master := m3u8.Master(b)

	list, err := master.GetVariantPlaylistByQuality(unit.Quality)
	if err != nil {
		return "", nil, err
	}

	parts := strings.Split(masterURL, "master.m3u8")
	listParts := strings.Split(list.URL, "/")

	basePath := parts[0] + listParts[0]
	playlistURL := parts[0] + list.URL

	b, err = c.fetchWithContext(ctx, http.MethodGet, playlistURL)
	if err != nil {
		return "", nil, err
	}

	playlist := m3u8.ParseMediaPlaylist(bytes.NewReader(b))
	playlist.TruncateSegments(unit.Start, unit.End)

	return basePath, &playlist, nil
}

func (c *Client) Download(ctx context.Context, u *Unit) error {
	// downloadVod is blocking, when done, notify
	err := c.downloadVod(ctx, u)

	c.notify(ProgressMessage{
		ID:    u.GetID(),
		Bytes: 0,
		Error: err,
		Done:  true,
	})

	return err
}

func (c *Client) downloadVod(ctx context.Context, unit *Unit) error {
	// MASTER URL NEEDS TO BE FETCHED AND PARSED SO WE CAN GET PLAYLIST QUALITY
	// TODO: WHOLE m3u8 PACKAGE NEEDS TO BE IMPROVED
	basePath, playlist, err := c.getMediaPlaylist(ctx, unit)
	if err != nil {
		return err
	}

	jobsChan := make(chan segmentJob)
	resultsChan := make(chan segmentJob)

	go func() {
		for i, seg := range playlist.Segments {
			if strings.HasSuffix(seg.URL, ".ts") {
				fullSegURL, _ := url.JoinPath(basePath, seg.URL)
				select {
				case <-ctx.Done():
					return
				case jobsChan <- segmentJob{
					index: i,
					url:   fullSegURL,
				}:
				}
			}
		}
	}()

	const maxWorkers = 8
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobsChan:
					if !ok {
						return
					}

					req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.url, nil)
					if err != nil {
						return
					}

					resp, err := c.httpClient.Do(req)
					if err != nil {
						job.err = err
					} else {
						data, err := io.ReadAll(resp.Body)
						job.err = err
						job.data = data
						resp.Body.Close()

						select {
						case <-ctx.Done():
							return
						case resultsChan <- job:
						}
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
		case <-ctx.Done():
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

					n, err := unit.W.Write(job.data)
					if err != nil {
						return fmt.Errorf("error writing segment: %v", err)
					}

					if ctx.Err() != nil {
						return ctx.Err()
					}

					msg := ProgressMessage{
						ID:    unit.GetID(),
						Error: unit.GetError(),
						Bytes: int64(n),
						Done:  false,
					}
					c.notify(msg)
				} else {
					break
				}
			}
		}
	}
}
