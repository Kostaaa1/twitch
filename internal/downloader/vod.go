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

	resp, err := httputil.Do(ctx, dl.http, variant.URL, http.MethodGet, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	playlist, err := m3u8.ParseMediaPlaylist(resp.Body, variant.URL)
	if err != nil {
		return nil, err
	}

	if unit.Start > 0 || unit.End > 0 {
		playlist.Truncate(unit.Start, unit.End)
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

	if err := u.setFileExt(list.Segments[0].URI); err != nil {
		return err
	}

	if list.Map != nil && list.Map.URI != "" {
		if err := dl.fetchDownload(ctx, u, buildSegURL(list.URL, list.Map.URI)); err != nil {
			return err
		}
	}

	currentChunk := atomic.Uint32{}
	depth := make(chan struct{}, dl.transfer.MaxReadAheadPerUnit)

	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < dl.transfer.MaxWorkersPerUnit; i++ {
		g.Go(func() error {
			for {
				chunkInx := int(currentChunk.Add(1) - 1)
				if chunkInx >= len(list.Segments) {
					return nil
				}

				select {
				case depth <- struct{}{}:
				case <-ctx.Done():
					return ctx.Err()
				}

				seg := list.Segments[chunkInx]
				segURL := buildSegURL(list.URL, seg.URI)

				body, err := dl.fetchSegment(ctx, u, segURL)
				if err != nil {
					return err
				}

				b, err := io.ReadAll(body)
				body.Close()
				if err != nil {
					return err
				}

				seg.Data <- b
				close(seg.Data)
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
			case b := <-list.Segments[i].Data:
				if err := dl.downloadBytes(u, b); err != nil {
					return err
				}
			}
		}
		return nil
	})

	return g.Wait()
}
