package m3u8

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type MediaPlaylist struct {
	Version        int64
	TargetDuration float64
	Timestamp      string
	PlaylistType   string
	ElapsedSecs    float64
	TotalSecs      float64
	Segments       []string
}

func (mp *MediaPlaylist) TruncateSegments(start, end time.Duration) error {
	segmentDuration := 10 // this is hardcoded value, maybe some segments are longer
	s := int(start.Seconds() / float64(segmentDuration))
	e := int(end.Seconds() / float64(segmentDuration))

	if s > len(mp.Segments) || e > len(mp.Segments) {
		totalSeconds := len(mp.Segments) * segmentDuration
		total := time.Duration(time.Second) * time.Duration(totalSeconds)
		return fmt.Errorf("invalid start/end parameters. You've choosen %s/%s but the video duration is %s", start, end, total)
	}

	if e == 0 {
		mp.Segments = mp.Segments[s:]
	} else {
		mp.Segments = mp.Segments[s:e]
	}

	return nil
}

func ParseMediaPlaylist(list []byte) MediaPlaylist {
	var mediaList MediaPlaylist
	lines := strings.Split(string(list), "\n")

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		k := parts[0]
		v := parts[1]

		switch {
		case k == "#EXT-X-VERSION":
			i, _ := strconv.ParseInt(v, 10, 64)
			mediaList.Version = i
		case k == "#EXT-X-TARGETDURATION":
			i, _ := strconv.ParseFloat(v, 64)
			mediaList.TargetDuration = i
		case k == "#EXT-X-PLAYLIST-TYPE":
			mediaList.PlaylistType = v
		case k == "#EXT-X-TWITCH-ELAPSED-SECS":
			i, _ := strconv.ParseFloat(v, 64)
			mediaList.ElapsedSecs = i
		case k == "#EXT-X-TWITCH-TOTAL-SECS":
			i, _ := strconv.ParseFloat(v, 64)
			mediaList.TotalSecs = i
		case strings.HasPrefix(k, "#ID3-EQUIV-TDTG"):
			mediaList.Timestamp = v
		case strings.HasPrefix(k, "#EXTINF"):
			mediaList.Segments = append(mediaList.Segments, lines[i+1])
			i++
		}
	}

	return mediaList
}
