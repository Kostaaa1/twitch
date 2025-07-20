package kick

import (
	"errors"
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
	MasterURL string
	Playlist  m3u8.MediaPlaylist

	Writer io.Writer

	Quality string
	Start   time.Duration
	End     time.Duration
	Error   error
}

func (unit *Unit) NotifyProgressChannel(msg spinner.ChannelMessage, progCh chan spinner.ChannelMessage) {
	if progCh == nil {
		return
	}
	if unit.Writer != nil {
		if file, ok := unit.Writer.(*os.File); ok && file != nil {
			if unit.Error != nil {
				os.Remove(file.Name())
				unit.Writer = nil
			}

			l := msg
			l.Text = file.Name()
			progCh <- l
		}
	}
}

func (unit *Unit) GetTitle() string {
	if f, ok := unit.Writer.(*os.File); ok && f != nil {
		return f.Name()
	}
	return "no title"
}

func (unit *Unit) GetError() error {
	return unit.Error
}

func (c *Client) NewUnit(w io.Writer, channel, vodID, quality string, start, end time.Duration) (*Unit, error) {
	masterURL, err := c.GetMasterPlaylistURL(channel, vodID)
	if err != nil {
		return nil, err
	}

	playlist, err := c.GetMediaPlaylist(masterURL, quality)
	if err != nil {
		return nil, err
	}

	return &Unit{
		MasterURL: &masterURL,
		Playlist:  playlist,
		Writer:    w,
		Quality:   quality,
		Start:     start,
		End:       end,
	}, nil
}

func (unit Unit) Close() error {
	return unit.Writer.(io.Closer).Close()
}

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

func (c *Client) Download() error {
	if unit.MasterURL == nil {
		return errors.New("masterURL is not set. It is used for extracting base URL for building segment URLs")
	}

	jobsChan := make(chan segmentJob, 16)
	resultsChan := make(chan segmentJob, 16)

	masterURL := "https://stream.kick.com/ivs/v1/196233775518/BqIVEMfsiezg/2025/7/12/19/40/Xozj3KO8N7BW/media/hls/master.m3u8"
	c.client.Get(masterURL)
	basePath := strings.TrimSuffix(masterURL, "master.m3u8")

	go func() {
		for i, seg := range unit.Playlist.Segments {
			if strings.HasSuffix(seg.URL, ".ts") {
				fullSegURL, _ := url.JoinPath(basePath, "1080", seg.URL)
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

					req, err := http.NewRequestWithContext(ctx, "GET", job.url, nil)
					if err != nil {
						job.err = err
					}

					resp, err := c.client.Do(req)
					if err != nil {
						job.err = err
					}

					b, err := io.ReadAll(resp.Body)
					resp.Body.Close()
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

					n, err := unit.Writer.Write(job.data)
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
