package kick

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/m3u8"
	"golang.org/x/sync/errgroup"
)

type segmentJob struct {
	index int
	url   string
	data  []byte
	err   error
}

func (c *Client) getMediaPlaylist(
	ctx context.Context,
	unit Unit,
) (string, *m3u8.MediaPlaylist, error) {
	masterURL, err := c.MasterPlaylistURL(unit.Channel, unit.UUID.String())
	if err != nil {
		return "", nil, err
	}

	res, err := c.cycletls.Do(masterURL, c.defaultCycleTLSOpts(), http.MethodGet)
	if err != nil {
		return "", nil, err
	}

	master := m3u8.Master(res.BodyBytes)

	list, err := master.GetVariantPlaylistByQuality(unit.Quality)
	if err != nil {
		return "", nil, err
	}

	parts := strings.Split(masterURL, "master.m3u8")
	listParts := strings.Split(list.URL, "/")

	basePath := parts[0] + listParts[0]
	playlistURL := parts[0] + list.URL

	res, err = c.cycletls.Do(playlistURL, c.defaultCycleTLSOpts(), http.MethodGet)
	if err != nil {
		return "", nil, err
	}

	playlist := m3u8.ParseMediaPlaylist(bytes.NewReader(res.BodyBytes))
	playlist.TruncateSegments(unit.Start, unit.End)

	return basePath, &playlist, nil
}

func (c *Client) Download(ctx context.Context, u Unit) error {
	err := c.downloadVOD(ctx, u)

	c.notify(Progress{
		ID:    u.GetID(),
		Bytes: 0,
		Error: err,
		Done:  true,
	})

	return err
}

func (c *Client) downloadVOD(ctx context.Context, unit Unit) error {
	u, playlist, err := c.getMediaPlaylist(ctx, unit)
	if err != nil {
		return err
	}

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

					if strings.HasSuffix(seg.URL, ".ts") {
						segmentURL, _ := url.JoinPath(u, seg.URL)

						res, err := c.cycletls.Do(segmentURL, c.defaultCycleTLSOpts(), http.MethodGet)
						if err != nil {
							return err
						}

						seg.Data <- io.NopCloser(bytes.NewReader(res.BodyBytes))
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
				n, err := io.Copy(unit.W, chunk)
				if err != nil {
					return err
				}

				c.notify(Progress{
					ID:    unit.GetID(),
					Error: unit.GetError(),
					Bytes: int64(n),
					Done:  false,
				})
				chunk.Close()
			}
		}
		return nil
	})

	g.Wait()

	return nil
}
