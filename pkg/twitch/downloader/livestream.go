package downloader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	segURLChan := make(chan string, 32)
	defer close(segURLChan)

	errCh := make(chan error, 1)
	defer close(errCh)

	go func() {
		dups := make(map[string]struct{})
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case tsURL, ok := <-segURLChan:
				if !ok {
					return
				}

				if _, ok := dups[tsURL]; ok {
					log.Fatalf("duplicate: %s", tsURL)
				}
				dups[tsURL] = struct{}{}

				if err := dl.download(ctx, unit, tsURL); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	lastSegmentURL := ""
	_ = lastSegmentURL
	pollCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-errCh:
			return err
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

			pollCount++
			s := bufio.NewScanner(resp.Body)

			lastPollURL := ""
			seenLastSegURL := false
			_ = seenLastSegURL

			for s.Scan() {
				line := s.Text()

				if strings.HasPrefix(line, "#EXTINF") {
					if strings.Contains(line, "Amazon") {
						continue
					}
					s.Scan()

					tsURL := s.Text()
					lastPollURL = tsURL
					if tsURL == lastSegmentURL {
						seenLastSegURL = true
					}

					fmt.Println("")
					fmt.Printf("\n#%d Poll\n", pollCount)
					fmt.Println("lastSegmentURL", lastSegmentURL)
					fmt.Println("seenLastSegURL", seenLastSegURL)
					fmt.Println("tsURL", tsURL)
					fmt.Println("downloaded", lastSegmentURL == "" || seenLastSegURL && tsURL != lastSegmentURL)

					if lastSegmentURL == "" || seenLastSegURL && tsURL != lastSegmentURL {
						segURLChan <- tsURL
					}
				}
			}

			lastSegmentURL = lastPollURL
			fmt.Println("polling ended, setting last segment for current poll", lastSegmentURL)
			resp.Body.Close()
		}
	}
}
