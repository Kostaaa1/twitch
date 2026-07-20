package downloader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/httputil"
)

func (dl *Downloader) recordLivestream(ctx context.Context, u *Unit) error {
	isLive, err := dl.gql.IsChannelLive(ctx, u.ID)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", u.ID)
	}

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
			case tsURL, ok := <-segURLChan:
				if !ok {
					return
				}
				if err := dl.fetchDownload(ctx, u, tsURL); err != nil {
					errCh <- err
					close(errCh)
					return
				}
			}
		}
	}()

	lastSegmentURL := ""

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-ticker.C:
			resp, err := httputil.Do(ctx, dl.http, list.URL, http.MethodGet, nil, nil)
			if err != nil {
				return err
			}

			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				return errors.New("playlist not found - channel is not live anymore")
			}

			s := bufio.NewScanner(resp.Body)

			lastPollURL := ""
			seenLastSegURL := false

			for s.Scan() {
				if s.Err() != nil {
					resp.Body.Close()
					return s.Err()
				}

				line := s.Text()

				if strings.HasPrefix(line, "#EXTINF") {
					// skipping ads..
					if strings.Contains(line, "Amazon") {
						continue
					}

					s.Scan()
					lastPollURL = s.Text()

					if lastPollURL == lastSegmentURL {
						seenLastSegURL = true
					}

					if lastSegmentURL == "" || seenLastSegURL && lastPollURL != lastSegmentURL {
						if u.ext == "" {
							if err := u.setFileExt(lastPollURL); err != nil {
								return err
							}
						}
						segURLChan <- lastPollURL
					}
				}
			}

			lastSegmentURL = lastPollURL
			resp.Body.Close()
		}
	}
}
