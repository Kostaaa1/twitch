package twitchdl

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/spinner"
)

// func getLastSegment(segments []string) string {
// 	for i := len(segments) - 1; i >= 0; i-- {
// 		segment := segments[i]
// 		if strings.HasPrefix(segment, "#EXTINF:") {
// 			return segment
// 		}
// 	}
// 	return ""
// }

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

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	count := 0
	maxCount := 1
	var byteBuf bytes.Buffer

	for range ticker.C {
		b, err := dl.fetch(variant.URL)
		if err != nil {
			msg := spinner.ChannelMessage{Error: errors.New("stream ended")}
			mu.NotifyProgressChannel(msg, dl.progressCh)
			return nil
		}

		segments := strings.Split(string(b), "\n")
		lastSegHeader := strings.TrimPrefix(segments[len(segments)-3], "#EXTINF:")

		// segments := strings.Split(string(b), "\n")
		// var lastSegURL string
		// var lastSegHeader string
		// for i := len(segments) - 1; i >= 0; i-- {
		// 	segment := segments[i]
		// 	if strings.HasPrefix(segment, "#EXTINF:") {
		// 		lastSegURL = segments[i+1]
		// 		lastSegHeader = segment
		// 		break
		// 	}
		// }

		// fmt.Println("- ", variant.URL)
		// fmt.Println("Index: ", id, " url: ", lastSegURL, " header: ", lastSegHeader, "Ad? ", strings.Contains(lastSegHeader, "Amazon"))

		if dl.config.SkipAds && strings.Contains(lastSegHeader, "Amazon") {
			msg := spinner.ChannelMessage{Message: "[Ad is running]", Bytes: 0}
			mu.NotifyProgressChannel(msg, dl.progressCh)
			continue
		}

		maxCount, _ = strconv.Atoi(strings.SplitN(lastSegHeader, ",", 2)[0])
		if maxCount <= 0 {
			maxCount = 1
		}

		if count == 0 {
			segmentBytes, _ := dl.fetch(lastSegURL)
			byteBuf.Reset()
			byteBuf.Write(segmentBytes)
		}

		segmentSize := byteBuf.Len() / maxCount
		start := count * segmentSize
		end := start + segmentSize
		if end > byteBuf.Len() {
			end = byteBuf.Len()
		}

		n, err := mu.Writer.Write(byteBuf.Bytes()[start:end])
		if err != nil {
			return err
		}

		msg := spinner.ChannelMessage{Bytes: int64(n)}
		mu.NotifyProgressChannel(msg, dl.progressCh)

		count++
		if count == maxCount {
			count = 0
		}
	}

	return nil
}
