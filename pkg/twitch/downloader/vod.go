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

func (dl *Downloader) getPlaylistsForUnit(ctx context.Context, unit Unit) (*m3u8.VariantPlaylist, *m3u8.MediaPlaylist, error) {
	master, err := dl.twClient.MasterPlaylistVOD(ctx, unit.ID)
	if err != nil {
		return nil, nil, err
	}

	variant, err := master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return nil, nil, err
	}

	playlist, err := dl.twClient.FetchAndParseMediaPlaylist(variant.URL, unit.Start, unit.End)
	if err != nil {
		return nil, nil, err
	}

	return variant, playlist, nil
}

func buildTSURL(baseURL, segment string) string {
	lastIndex := strings.LastIndex(baseURL, "/")
	return fmt.Sprintf("%s/%s", baseURL[:lastIndex], segment)
}

func (dl *Downloader) fetchSegmentWithRetry(ctx context.Context, u string) (io.ReadCloser, error) {
	data, status, err := dl.fetch(ctx, u)
	if status == http.StatusForbidden {
		switch {
		case strings.Contains(u, "unmuted"):
			u = strings.Replace(u, "-unmuted", "-muted", 1)
			data, _, err = dl.fetch(ctx, u)
		case strings.Contains(u, "muted"):
			u = strings.Replace(u, "-muted", "", 1)
			data, _, err = dl.fetch(ctx, u)
		}
	}

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (dl *Downloader) downloadVOD(ctx context.Context, unit Unit) error {
	variant, playlist, _ := dl.getPlaylistsForUnit(ctx, unit)

	g, ctx := errgroup.WithContext(ctx)
	currentChunk := atomic.Uint32{}

	for i := 0; i < 8; i++ {
		g.Go(func() error {
			for {
				chunkInx := int(currentChunk.Add(1) - 1)
				if chunkInx >= len(playlist.Segments) {
					return nil
				}

				seg := playlist.Segments[chunkInx]

				if strings.HasSuffix(seg.URL, ".ts") {
					fullSegURL := buildTSURL(variant.URL, seg.URL)

					reader, err := dl.fetchSegmentWithRetry(ctx, fullSegURL)
					if err != nil {
						return err
					}

					seg.Data <- reader
					close(seg.Data)
				}
			}
		})
	}

	g.Go(func() error {
		for i := 0; i < len(playlist.Segments); i++ {
			select {
			case <-ctx.Done():
				return nil
			case chunk, ok := <-playlist.Segments[i].Data:
				if !ok {
					continue
				}
				n, err := io.Copy(unit.Writer, chunk)
				if err != nil {
					return err
				}
				dl.notify(Progress{
					ID:    unit.GetID(),
					Err:   unit.Error,
					Bytes: n,
				})
				chunk.Close()
			}
		}

		return nil
	})

	return g.Wait()
}

func (unit Unit) StreamVOD(ctx context.Context, dl *Downloader) error {
	master, err := dl.twClient.MasterPlaylistVOD(ctx, unit.ID)
	if err != nil {
		return err
	}
	variant, err := master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return err
	}
	resp, err := dl.twClient.HttpClient().Get(variant.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	playlist, err := m3u8.ParseMediaPlaylist(resp.Body)
	if err != nil {
		return err
	}
	playlist.Truncate(unit.Start, unit.End)

	for _, seg := range playlist.Segments {
		if strings.HasSuffix(seg.URL, ".ts") {
			lastIndex := strings.LastIndex(variant.URL, "/")
			fullSegURL := fmt.Sprintf("%s/%s", variant.URL[:lastIndex], seg.URL)

			reader, _, err := dl.fetch(ctx, fullSegURL)
			if err != nil {
				return err
			}

			_, err = io.Copy(unit.Writer, reader)
			if err != nil {
				return err
			}

			// msg := spinner.Message{ID: unit.GetID(), Bytes: n}
			// dl.NotifyProgressChannel(msg, unit)
		}
	}

	return nil
}
