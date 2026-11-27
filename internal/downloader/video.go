package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/Kostaaa1/twitch/internal/downloader/m3u8"
	"github.com/Kostaaa1/twitch/internal/httputil"
	"golang.org/x/sync/errgroup"
)

func (dl *Downloader) mediaPlaylistVideo(ctx context.Context, unit *Unit) (*m3u8.MediaPlaylist, error) {
	master, err := dl.usher.MasterPlaylistVideo(ctx, unit.ID)
	if err != nil {
		return nil, err
	}

	variant, err := master.VariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return nil, err
	}

	resp, err := httputil.Do(ctx, dl.http, variant.Source, http.MethodGet, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	playlist, err := m3u8.ParseMediaPlaylist(resp.Body, variant.Source)
	if err != nil {
		return nil, err
	}

	if err := playlist.Truncate(unit.Start, unit.End); err != nil {
		return nil, err
	}

	return playlist, nil
}

func buildSegURL(playlistURL, path string) string {
	lastIndex := strings.LastIndex(playlistURL, "/")
	return fmt.Sprintf("%s/%s", playlistURL[:lastIndex], path)
}

func (dl *Downloader) downloadVideo(ctx context.Context, u *Unit) error {
	list, err := dl.mediaPlaylistVideo(ctx, u)
	if err != nil {
		return err
	}
	if list.Map != nil && list.Map.URI != "" {
		mp4segURL := buildSegURL(list.Source, list.Map.URI)
		if err := dl.fetchDownload(ctx, u, mp4segURL); err != nil {
			return err
		}
	}

	// allocate data channels
	results := make([]chan []byte, len(list.Segments))
	for i := 0; i < len(list.Segments); i++ {
		results[i] = make(chan []byte, 1)
	}

	if u.ext == "" {
		if err := u.setFileExt(list.Segments[0].URI); err != nil {
			return err
		}
	}

	currentChunk := atomic.Uint32{}
	depth := make(chan struct{}, dl.transfer.MaxReadAheadPerUnit)

	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < dl.transfer.MaxWorkersPerUnit; i++ {
		g.Go(func() error {
			for {
				chunkIdx := int(currentChunk.Add(1) - 1)
				if chunkIdx >= len(list.Segments) {
					return nil
				}

				select {
				case depth <- struct{}{}:
				case <-ctx.Done():
					return ctx.Err()
				}

				seg := list.Segments[chunkIdx]

				segURL := buildSegURL(list.Source, seg.URI)
				body, err := dl.fetchSegment(ctx, u, segURL)
				if err != nil {
					return err
				}

				b, err := io.ReadAll(body)
				body.Close()
				if err != nil {
					return err
				}

				results[chunkIdx] <- b
				close(results[chunkIdx])
			}
		})
	}

	g.Go(func() error {
		for i := 0; i < len(list.Segments); i++ {
			select {
			case <-depth:
			case <-ctx.Done():
				return ctx.Err()
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case b := <-results[i]:
				if err := dl.downloadBytes(ctx, u, b); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return g.Wait()
}
