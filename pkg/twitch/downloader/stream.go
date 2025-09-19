package downloader

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

type segmentHistory struct {
	seen map[string]struct{}
	list []string
	max  int
}

func (h *segmentHistory) Add(url string) {
	if _, ok := h.seen[url]; ok {
		return
	}
	h.seen[url] = struct{}{}
	h.list = append(h.list, url)

	if len(h.list) > h.max {
		old := h.list[0]
		h.list = h.list[:1]
		delete(h.seen, old)
	}
}

func (h *segmentHistory) Seen(url string) bool {
	if _, ok := h.seen[url]; ok {
		return true
	}
	return false
}

func (dl *Downloader) recordStream(ctx context.Context, unit Unit) error {
	isLive, err := dl.twClient.IsChannelLive(unit.ID)
	if err != nil {
		return err
	}

	if !isLive {
		return fmt.Errorf("%s is offline", unit.ID)
	}

	master, err := dl.twClient.MasterPlaylistStream(unit.ID)
	if err != nil {
		return err
	}

	variant, err := master.GetVariantPlaylistByQuality(unit.Quality.String())
	if err != nil {
		return err
	}

	segHist := segmentHistory{
		seen: make(map[string]struct{}),
		list: make([]string, 0),
		max:  500,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
			// TODO: we do not really need to fetch segments all the time.
			reader, _, err := dl.fetch(ctx, variant.URL)
			if err != nil {
				return err
			}
			defer reader.Close()

			b, err := io.ReadAll(reader)
			if err != nil {
				return err
			}

			lines := strings.Split(string(b), "\n")

			for i := 0; i < len(lines)-1; i++ {
				select {
				case <-ctx.Done():
					return nil
				default:
					line := lines[i]
					if strings.HasPrefix(line, "#EXTINF") {
						if strings.Contains(line, "Amazon") {
							continue
						}

						segURL := lines[i+1]
						if segHist.Seen(segURL) {
							continue
						}

						// if segmentURL has *-unmuted.ts, this will give 403
						// if strings.Contains(segURL, "unmuted") {
						// 	segURL = strings.Replace(segURL, "unmuted", "muted", 1)
						// }

						reader, _, err := dl.fetch(ctx, segURL)
						if err != nil {
							return err
						}

						n, err := io.Copy(unit.Writer, reader)
						if err != nil {
							return err
						}

						// msg := spinner.Message{ID: unit.GetID(), Bytes: int64(n)}
						// dl.NotifyProgressChannel(msg, unit)
						dl.notify(ProgressMessage{
							ID:    unit.GetID(),
							Err:   unit.Error,
							Bytes: n,
							Done:  false,
						})

						segHist.Add(segURL)
					}
				}
			}
		}
	}
}
