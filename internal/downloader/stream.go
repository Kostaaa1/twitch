package downloader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/httputil"
)

func newSegments(s []string, last string) ([]string, string) {
	if len(s) == 0 {
		return nil, last
	}

	idx := slices.Index(s, last)
	if idx == -1 {
		return s, s[len(s)-1]
	}

	if idx+1 >= len(s) {
		return nil, last
	}

	send := s[idx+1:]
	return send, send[len(send)-1]
}

func (dl *Downloader) recordLivestream(ctx context.Context, u *Unit) error {
	master, err := dl.usher.MasterPlaylistStream(ctx, u.ID)
	if err != nil {
		return err
	}

	list, err := master.VariantPlaylistByQuality(u.Quality.String())
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	segURLChan := make(chan string, 32)
	defer close(segURLChan)

	errCh := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case segURL, ok := <-segURLChan:
				if !ok {
					return
				}
				if err := dl.fetchDownload(ctx, u, segURL); err != nil {
					errCh <- err
					close(errCh)
					return
				}
			}
		}
	}()

	lastSegURL := ""

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-ticker.C:
			resp, err := httputil.Do(ctx, dl.http, list.Source, http.MethodGet, nil, nil)
			if err != nil {
				return err
			}
			code := resp.StatusCode
			if code == http.StatusNotFound {
				resp.Body.Close()
				return errors.New("playlist not found - channel is not currently live")
			}
			if code < 200 || code >= 300 {
				resp.Body.Close()
				return fmt.Errorf("non success code for livestream playlist: playlist=%s code=%d", list.Source, code)
			}

			s := bufio.NewScanner(resp.Body)
			segments := make([]string, 0)

			for s.Scan() {
				if s.Err() != nil {
					resp.Body.Close()
					return s.Err()
				}
				line := s.Text()
				if strings.HasPrefix(line, "#EXTINF") {
					if strings.Contains(line, "Amazon") {
						continue
					}
					s.Scan()
					segments = append(segments, line)
				}
			}

			seen, last := newSegments(segments, lastSegURL)
			if last != "" {
				for _, segURL := range seen {
					if u.ext == "" {
						if err := u.setFileExt(segURL); err != nil {
							return err
						}
					}
					segURLChan <- segURL
				}
				lastSegURL = last
			}

			resp.Body.Close()
		}
	}
}
