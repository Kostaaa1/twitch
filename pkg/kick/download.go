package kick

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kostaaa1/twitch/internal/downloader/m3u8"
	"golang.org/x/sync/errgroup"
)

func (c *Downloader) mediaPlaylist(ctx context.Context, unit *Unit) (string, *m3u8.MediaPlaylist, error) {
	masterURL, err := c.MasterPlaylistURL(unit.Channel, unit.UUID.String())
	if err != nil {
		return "", nil, err
	}

	fmt.Println("MASTER:", masterURL)

	res, err := c.cycletls.Do(masterURL, c.defaultCycleTLSOpts(), http.MethodGet)
	if err != nil {
		return "", nil, err
	}

	master := m3u8.Master(res.BodyBytes)

	list, err := master.VariantPlaylistByQuality(unit.Quality)
	if err != nil {
		return "", nil, err
	}

	parts := strings.Split(masterURL, "master.m3u8")
	listParts := strings.Split(list.Source, "/")

	basePath := parts[0] + listParts[0]
	playlistURL := parts[0] + list.Source

	res, err = c.cycletls.Do(playlistURL, c.defaultCycleTLSOpts(), http.MethodGet)
	if err != nil {
		return "", nil, err
	}

	playlist, err := m3u8.ParseMediaPlaylist(bytes.NewReader(res.BodyBytes), playlistURL)
	if err != nil {
		return "", nil, err
	}
	if err := playlist.Truncate(unit.Start, unit.End); err != nil {
		return "", nil, err
	}

	return basePath, playlist, nil
}

func (c *Downloader) Download(ctx context.Context, u *Unit) error {
	err := c.downloadVOD(ctx, u)
	c.notify(u, 0)
	return err
}

func (c *Downloader) downloadVOD(ctx context.Context, unit *Unit) error {
	fmt.Println("DOWNLOADING VOD:", &unit)

	u, playlist, err := c.mediaPlaylist(ctx, unit)
	if err != nil {
		fmt.Println("FAILED TO GET MEDIAPLALISY", err)
		return err
	}

	fmt.Println("DOWNLOADIN VODJjA", u)
	fmt.Println("DOWNLOADIN VODJjA", &playlist)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	g.Go(func() error {
		for _, seg := range playlist.Segments {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				seg := seg

				g.Go(func() error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}

					if strings.HasSuffix(seg.URI, ".ts") {
						segURL, _ := url.JoinPath(u, seg.URI)

						res, err := c.cycletls.Do(
							segURL,
							c.defaultCycleTLSOpts(),
							http.MethodGet,
						)
						if err != nil {
							return err
						}

						seg.Data <- res.BodyBytes
						close(seg.Data)
					}
					return nil
				})
			}
		}
		return nil
	})

	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		for i := 0; i < len(playlist.Segments); i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case chunk := <-playlist.Segments[i].Data:
				n, err := unit.W.Write(chunk)
				if err != nil {
					return err
				}
				c.notify(unit, int64(n))
			}
		}
		return nil
	})

	g.Wait()

	return nil
}
