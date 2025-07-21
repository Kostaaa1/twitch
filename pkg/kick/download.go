package kick

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/pkg/m3u8"
	"github.com/Kostaaa1/twitch/pkg/spinner"
)

type Unit struct {
	URL     string
	W       io.Writer
	Start   time.Duration
	End     time.Duration
	Quality string
	Error   error
}

func (u Unit) GetError() error {
	return u.Error
}

func (u Unit) GetTitle() string {
	if f, ok := u.W.(*os.File); ok && f != nil {
		return f.Name()
	}
	return ""
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.W.(*os.File); ok && f != nil {
		if u.Error != nil {
			os.Remove(f.Name())
		}
		return f.Close()
	}
	return nil
}

func (unit *Unit) NotifyProgressChannel(msg spinner.ChannelMessage, progressCh chan spinner.ChannelMessage) {
	if progressCh == nil {
		return
	}
	if unit.W != nil {
		if file, ok := unit.W.(*os.File); ok && file != nil {
			if unit.Error != nil {
				os.Remove(file.Name())
				unit.W = nil
			}
			l := msg
			l.Text = file.Name()
			progressCh <- l
		}
	}
}

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

func (c *Client) Download(ctx context.Context, unit Unit) error {
	masterURL, err := c.MasterPlaylistURL(unit.URL)
	if err != nil {
		return fmt.Errorf("failed to get m3u8 master URL: %s", err.Error())
	}

	basePath := strings.TrimSuffix(masterURL, "master.m3u8")
	playlistURL := basePath + unit.Quality + "/playlist.m3u8"

	res, err := c.client.Get(playlistURL)
	if err != nil {
		return fmt.Errorf("failed to fetch media playlist: %s", err.Error())
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to io.ReadAll(): %s", err.Error())
	}

	playlist := m3u8.ParseMediaPlaylist(b)
	// playlist.TruncateSegments(start, end)

	jobsChan := make(chan segmentJob, 16)
	resultsChan := make(chan segmentJob, 16)

	go func() {
		for i, seg := range playlist.Segments {
			if strings.HasSuffix(seg.URL, ".ts") {
				fullSegURL, _ := url.JoinPath(basePath, unit.Quality, seg.URL)
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
		close(jobsChan)
	}()

	const maxWorkers = 16
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

					req, err := http.NewRequestWithContext(ctx, "GET", job.url, nil)
					if err != nil {
						job.err = err
					}
					// setDefaultHeaders(req)

					res, err := c.client.Do(req)
					if err != nil {
						fmt.Println(err)
						job.err = err
					}

					b, err := io.ReadAll(res.Body)
					res.Body.Close()
					if err != nil {
						job.err = err
					}
					job.data = b
					job.err = nil

					select {
					case <-ctx.Done():
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
						log.Fatal(err)
					}

					msg := spinner.ChannelMessage{Bytes: int64(n)}
					unit.NotifyProgressChannel(msg, c.progCh)
				} else {
					break
				}
			}
		}
	}
}
