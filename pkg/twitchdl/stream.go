package twitchdl

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/spinner"
)

type segmentHist struct {
	seen map[string]struct{}
	list []string
	max  int
}

func (h *segmentHist) Add(url string) {
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

func (h *segmentHist) Seen(url string) bool {
	if _, ok := h.seen[url]; ok {
		return true
	}
	return false
}

func (mu *Unit) recordStream(dl *Downloader) error {
	isLive, err := dl.TWApi.IsChannelLive(mu.ID)
	if err != nil {
		return err
	}
	if !isLive {
		return fmt.Errorf("%s is offline", mu.ID)
	}

	master, err := dl.TWApi.MasterPlaylistStream(mu.ID)
	if err != nil {
		return err
	}

	variant, err := master.GetVariantPlaylistByQuality(mu.Quality.String())
	if err != nil {
		return err
	}

	segHist := segmentHist{
		seen: make(map[string]struct{}),
		list: make([]string, 0),
		max:  500,
	}

	for {
		b, err := dl.fetch(variant.URL)
		if err != nil {
			msg := spinner.ChannelMessage{Error: errors.New("stream ended")}
			mu.NotifyProgressChannel(msg, dl.progressCh)
			return err
		}

		lines := strings.Split(string(b), "\n")

		for i := 0; i < len(lines)-1; i++ {
			line := lines[i]
			if strings.HasPrefix(line, "#EXTINF") {
				if dl.config.SkipAds && strings.Contains(line, "Amazon") {
					msg := spinner.ChannelMessage{Message: "[Ad is running]", Bytes: 0}
					mu.NotifyProgressChannel(msg, dl.progressCh)
					continue
				}

				segURL := lines[i+1]
				if segHist.Seen(segURL) {
					continue
				}
				segmentBytes, _ := dl.fetch(segURL)

				n, err := mu.Writer.Write(segmentBytes)
				if err != nil {
					log.Fatal(err)
				}

				msg := spinner.ChannelMessage{Bytes: int64(n)}
				mu.NotifyProgressChannel(msg, dl.progressCh)

				segHist.Add(segURL)
			}
		}
		time.Sleep(1 * time.Second)
	}
}
