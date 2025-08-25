package kick

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/m3u8"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
)

type QualityType int

const (
	QualityBest QualityType = iota
	Quality1080p60
	Quality720p60
	Quality480p30
	Quality360p30
	Quality160p30
	QualityWorst
	QualityAudioOnly
)

type Unit struct {
	URL     string
	Start   time.Duration
	End     time.Duration
	Quality downloader.QualityType
	// used for file creation
	Title string
	W     io.Writer
	Error error
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

//////////////////

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
	playlistURL := basePath + unit.Quality.String() + "/playlist.m3u8"

	res, err := c.client.Get(playlistURL)
	if err != nil {
		return fmt.Errorf("failed to fetch media playlist: %s", err.Error())
	}
	defer res.Body.Close()

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

					resp, err := c.client.Get(job.url)
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

					msg := spinner.ChannelMessage{Bytes: int64(n)}
					unit.NotifyProgressChannel(msg, c.progCh)
				} else {
					break
				}
			}
		}
	}
}
