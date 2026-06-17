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

	"github.com/Kostaaa1/twitch/internal/downloader/m3u8"
)

func (dl *Downloader) download(ctx context.Context, unit Unit, tsURL string) error {
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
	isLive, err := dl.twClient.Gql.IsChannelLive(ctx, unit.ID)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", unit.ID)
	}

	fmt.Println("IS CHANNEL LIVE:", isLive)

	b, err := dl.MasterPlaylistStream(ctx, unit.ID)
	if err != nil {
		return err
	}
	fmt.Println("MASTER BUYTE:", b)
	fmt.Println("MASTER:", string(b))

	master := m3u8.Master(b)

	variant, err := master.VariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return err
	}
	fmt.Println("Variant", variant)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	segURLChan := make(chan string, 32)
	defer close(segURLChan)

	errCh := make(chan error, 1)
	defer close(errCh)

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

	lastSegmentURL := ""

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

			s := bufio.NewScanner(resp.Body)

			lastPollURL := ""
			seenLastSegURL := false

			for s.Scan() {
				if s.Err() != nil {
					log.Fatal(err)
				}

				line := s.Text()

				if strings.HasPrefix(line, "#EXTINF") {
					if strings.Contains(line, "Amazon") {
						continue
					}
					s.Scan()

					lastPollURL = s.Text()
					if lastPollURL == lastSegmentURL {
						seenLastSegURL = true
					}

					if lastSegmentURL == "" || seenLastSegURL && lastPollURL != lastSegmentURL {
						segURLChan <- lastPollURL
					}
				}
			}

			lastSegmentURL = lastPollURL
			resp.Body.Close()
		}
	}
}
