package downloader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (dl *Downloader) download(ctx context.Context, unit Unit, tsURL string) error {
	fmt.Println("downloading", tsURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tsURL, nil)
	if err != nil {
		return err
	}

	resp, err := dl.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	n, err := io.Copy(unit.Writer, resp.Body)
	if err != nil {
		return err
	}

	dl.notify(Progress{
		ID:    unit.GetID(),
		Err:   unit.Error,
		Bytes: n,
	})

	return nil
}

func (dl *Downloader) recordLivestream(ctx context.Context, unit Unit) error {
	isLive, err := dl.twClient.IsChannelLive(ctx, unit.ID)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", unit.ID)
	}

	master, err := dl.twClient.MasterPlaylistStream(ctx, unit.ID)
	if err != nil {
		return err
	}
	variant, err := master.VariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return err
	}

	// producer
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	segURLChan := make(chan string, 32)
	defer close(segURLChan)

	errCh := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case tsURL, ok := <-segURLChan:
				if !ok {
					return
				}
				if err := dl.download(ctx, unit, tsURL); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	lastSegURL := ""
	firstPollPassed := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, variant.URL, nil)
			if err != nil {
				return err
			}
			resp, err := dl.http.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode == http.StatusNotFound {
				return errors.New("playlist not found - channel is not live anymore")
			}

			s := bufio.NewScanner(resp.Body)
			foundSegURL := false

			for s.Scan() {
				line := s.Text()

				if strings.HasPrefix(line, "#EXTINF") {
					// if strings.Contains(line, "Amazon") {
					// 	fmt.Println("Skipping AD")
					// 	continue
					// }

					if !s.Scan() {
						break
					}
					tsURL := s.Text()

					if lastSegURL == tsURL {
						foundSegURL = true
						continue
					}

					if firstPollPassed && !foundSegURL {
						continue
					}

					lastSegURL = tsURL
					segURLChan <- tsURL
				}
			}

			firstPollPassed = true
			resp.Body.Close()
		}
	}
}
