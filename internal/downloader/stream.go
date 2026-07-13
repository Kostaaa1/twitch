package downloader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/downloader/m3u8"
	"github.com/Kostaaa1/twitch/internal/httputil"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
)

func (dl *Downloader) MasterPlaylistStream(ctx context.Context, channel string) ([]byte, error) {
	tok, err := dl.gql.StreamPlaybackAccessToken(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestream credentials: %w", err)
	}

	url := fmt.Sprintf("%s/api/channel/hls/%s.m3u8?token=%s&sig=%s&allow_audio_only=true&allow_source=true", gql.UsherURL, channel, tok.Value, tok.Signature)

	b, _, err := httputil.DoBytes(
		ctx,
		dl.http,
		url,
		http.MethodGet,
		nil,
		nil,
	)

	return b, nil
}

func (dl *Downloader) recordLivestream(ctx context.Context, unit *Unit) error {
	isLive, err := dl.gql.IsChannelLive(ctx, unit.ID)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", unit.ID)
	}

	b, err := dl.MasterPlaylistStream(ctx, unit.ID)
	if err != nil {
		return err
	}

	variant, err := m3u8.Master(b).VariantPlaylistByQuality(unit.Quality.String())
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
				if err := dl.segmentFetchDownload(ctx, unit, tsURL); err != nil {
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
			resp, err := httputil.Do(ctx, dl.http, variant.URL, http.MethodGet, nil, nil)
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
