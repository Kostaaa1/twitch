package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/Kostaaa1/twitch/pkg/twitch/m3u8"
	"golang.org/x/sync/errgroup"
)

func (dl *Downloader) mediaPlaylistForUnit(ctx context.Context, unit Unit) (*m3u8.MediaPlaylist, error) {
	// Get master playlist for VOD by its ID
	master, err := dl.twClient.MasterPlaylistVOD(ctx, unit.ID)
	if err != nil {
		return nil, err
	}
	// Get playlist URL by specified quality
	variant, err := master.VariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return nil, err
	}
	// Fetch playlist
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, variant.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := dl.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Parse it
	playlist, err := m3u8.ParseMediaPlaylist(resp.Body, variant.URL)
	if err != nil {
		return nil, err
	}
	// Truncate
	playlist.Truncate(unit.Start, unit.End)

	return playlist, nil
}

func (dl *Downloader) downloadVOD(ctx context.Context, unit Unit) error {
	playlist, err := dl.mediaPlaylistForUnit(ctx, unit)
	if err != nil {
		return err
	}

	workerCount := 4

	g, ctx := errgroup.WithContext(ctx)

	currentChunk := atomic.Uint32{}

	for i := 0; i < workerCount; i++ {
		g.Go(func() error {
			for {
				chunkInx := int(currentChunk.Add(1) - 1)
				if chunkInx >= len(playlist.Segments) {
					return nil
				}

				seg := playlist.Segments[chunkInx]

				if strings.HasSuffix(seg.URL, ".ts") {
					lastIndex := strings.LastIndex(playlist.URL, "/")
					tsURL := fmt.Sprintf("%s/%s", playlist.URL[:lastIndex], seg.URL)

					req, err := http.NewRequestWithContext(ctx, http.MethodGet, tsURL, nil)
					if err != nil {
						return err
					}
					resp, err := dl.http.Do(req)
					if err != nil {
						return err
					}

					if resp.StatusCode == http.StatusForbidden {
						switch {
						case strings.Contains(tsURL, "unmuted"):
							tsURL = strings.Replace(tsURL, "-unmuted", "-muted", 1)
						case strings.Contains(tsURL, "muted"):
							tsURL = strings.Replace(tsURL, "-muted", "", 1)
						default:
							return fmt.Errorf("forbidden for segment: %s", tsURL)
						}

						req, err := http.NewRequestWithContext(ctx, http.MethodGet, tsURL, nil)
						if err != nil {
							return err
						}
						resp, err = dl.http.Do(req)
						if err != nil {
							return err
						}
					}

					seg.Data <- resp.Body
					close(seg.Data)
				}
			}
		})
	}

	g.Go(func() error {
		for i := 0; i < len(playlist.Segments); i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case chunk := <-playlist.Segments[i].Data:
				err := func() error {
					defer chunk.Close()

					n, err := io.Copy(unit.Writer, chunk)
					if err != nil {
						return err
					}

					dl.notify(Progress{
						ID:    unit.GetID(),
						Err:   unit.Error,
						Bytes: n,
					})

					return nil
				}()

				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return g.Wait()
}

// NOT USED
func (unit Unit) StreamVOD(ctx context.Context, dl *Downloader) error {
	playlist, err := dl.mediaPlaylistForUnit(ctx, unit)
	if err != nil {
		return err
	}

	for _, seg := range playlist.Segments {
		if strings.HasSuffix(seg.URL, ".ts") {
			lastIndex := strings.LastIndex(playlist.URL, "/")
			tsURL := fmt.Sprintf("%s/%s", playlist.URL[:lastIndex], seg.URL)

			if err := dl.download(ctx, unit, tsURL); err != nil {
				return err
			}
		}
	}

	return nil
}
