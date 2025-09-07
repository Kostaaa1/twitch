package downloader

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/spinner"
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

func (dl *Downloader) recordStream(unit Unit) error {
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
		case <-dl.ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
			// TODO: improve this so it does not use io.ReadAll()
			b, err := dl.fetch(variant.URL)
			if err != nil {
				msg := spinner.Message{Error: errors.New("stream ended")}
				unit.NotifyProgressChannel(msg, dl.progCh)
				return err
			}

			lines := strings.Split(string(b), "\n")
			for i := 0; i < len(lines)-1; i++ {
				select {
				case <-dl.ctx.Done():
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

						segmentBytes, err := dl.fetch(segURL)
						if err != nil {
							return err
						}

						n, err := unit.Writer.Write(segmentBytes)
						if err != nil {
							log.Fatal(err)
						}

						msg := spinner.Message{Bytes: int64(n)}
						unit.NotifyProgressChannel(msg, dl.progCh)
						segHist.Add(segURL)
					}
				}
			}
		}
	}
}
