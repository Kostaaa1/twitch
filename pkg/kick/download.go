package kick

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/m3u8"
	"github.com/Kostaaa1/twitch/pkg/spinner"
)

type QualityType int

const (
	Quality1080p QualityType = iota
	Quality720p60
	Quality720p30
	Quality480p30
	Quality360p30
	Quality160p30
)

func (qt QualityType) String() string {
	switch qt {
	case Quality1080p:
		return "1080p"
	case Quality720p60:
		return "720p"
	case Quality720p30:
		return "720p"
	case Quality480p30:
		return "480p"
	case Quality360p30:
		return "360p"
	case Quality160p30:
		return "160p"
	default:
		return ""
	}
}

type Unit struct {
	URL     string
	Start   time.Duration
	End     time.Duration
	Quality QualityType
	// used for file creation
	Title string
	W     io.Writer
	Error error
}

// Satisfies spinner.UnitProvider
func (u Unit) GetError() error {
	return u.Error
}

func (u Unit) GetID() any {
	return u.Title
}

func (u Unit) GetTitle() string {
	if f, ok := u.W.(*os.File); ok && f != nil {
		return f.Name()
	}
	return ""
}

func (u *Unit) CreateFile(output string) error {
	if output == "" {
		return errors.New("output path not provided")
	}
	ext := "mp4"
	if strings.HasPrefix(u.Quality.String(), "audio") {
		ext = "mp3"
	}
	u.W, u.Error = fileutil.CreateFile(output, u.Title, ext)
	return nil
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

func (unit *Unit) NotifyProgressChannel(msg spinner.Message, ch chan spinner.Message) {
	if unit.W == nil || ch == nil {
		return
	}
	msg.ID = unit.Title
	ch <- msg
}

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

func (c *Client) Download(ctx context.Context, unit Unit) error {
	// MASTER URL NEEDS TO BE FETCHED AND PARSED SO WE CAN GET PLAYLIST QUALITY
	// TODO: WHOLE m3u8 PACKAGE NEEDS TO BE IMPROVED
	masterURL, err := c.MasterPlaylistURL(unit.URL)
	if err != nil {
		return fmt.Errorf("failed to get m3u8 master URL: %s", err.Error())
	}
	basePath := strings.TrimSuffix(masterURL, "master.m3u8")
	playlistURL := basePath + unit.Quality.String() + "/playlist.m3u8"

	res, err := c.httpClient.Get(playlistURL)
	if err != nil {
		return fmt.Errorf("failed to fetch media playlist: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusForbidden {
		return fmt.Errorf("access to the stream is forbidden (403)")
	}

	playlist := m3u8.ParseMediaPlaylist(res.Body)
	playlist.TruncateSegments(unit.Start, unit.End)

	jobsChan := make(chan segmentJob)
	resultsChan := make(chan segmentJob)

	go func() {
		for i, seg := range playlist.Segments {
			if strings.HasSuffix(seg.URL, ".ts") {
				fullSegURL, _ := url.JoinPath(basePath, unit.Quality.String(), seg.URL)
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

					resp, err := c.httpClient.Get(job.url)
					if err != nil {
						job.err = err
					} else {
						// Avoid using of the io.ReadAll???????
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

					msg := spinner.Message{Bytes: int64(n)}
					unit.NotifyProgressChannel(msg, c.progCh)
				} else {
					break
				}
			}
		}
	}
}
