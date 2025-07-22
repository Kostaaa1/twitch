package m3u8

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"
)

type Segment struct {
	URL      string
	Duration time.Duration
}

type MediaPlaylist struct {
	Version         int64
	TargetDuration  float64
	Timestamp       string
	PlaylistType    string
	ElapsedSecs     float64
	TotalSecs       float64
	SegmentDuration time.Duration
	Segments        []Segment
}

func (mp *MediaPlaylist) TruncateSegments(start, end time.Duration) {
	if start < 0 || end < 0 || start == end || start > end {
		return
	}

	// figure out the way to skip first portion of segments based on start, maybe use
	total := time.Duration(0)
	startIndex := 0
	endIndex := 0

	for i, seg := range mp.Segments {
		if total <= start {
			total += seg.Duration
			startIndex = i
			continue
		}
		if total <= end {
			total += seg.Duration
			endIndex = i
			continue
		}
		break
	}

	mp.Segments = mp.Segments[startIndex : endIndex+1]
}

func ParseMediaPlaylist(r io.Reader) MediaPlaylist {
	var mediaList MediaPlaylist
	reader := bufio.NewReader(r)

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("failed to parsed media playlist: %s\n", err.Error())
			break
		}

		id := bytes.IndexByte(line, ':')
		if id == -1 {
			continue
		}

		key := string(line[:id])
		v := string(line[id+1:])

		switch key {
		case "#EXT-X-VERSION":
			value, _ := strconv.ParseInt(v, 10, 64)
			mediaList.Version = value
		case "#EXT-X-TARGETDURATION":
			value, _ := strconv.ParseFloat(v, 64)
			mediaList.TargetDuration = value
		case "#EXT-X-PLAYLIST-TYPE":
			mediaList.PlaylistType = v
		case "#EXT-X-TWITCH-ELAPSED-SECS":
			value, _ := strconv.ParseFloat(v, 64)
			mediaList.ElapsedSecs = value
		case "#EXT-X-TWITCH-TOTAL-SECS":
			value, _ := strconv.ParseFloat(v, 64)
			mediaList.TotalSecs = value
		case "#ID3-EQUIV-TDTG":
			mediaList.Timestamp = v
		case "#EXTINF":
			trimmed := v[:len(v)-1]
			seconds, _ := strconv.ParseFloat(trimmed, 64)
			duration := time.Duration(seconds * float64(time.Second))

			segmentURL, _, err := reader.ReadLine()
			if err != nil {
				log.Fatalf("failed to read next line: %s", err)
				break
			}

			seg := Segment{URL: string(segmentURL), Duration: duration}
			mediaList.Segments = append(mediaList.Segments, seg)
		}
	}

	return mediaList
}

// func ParseMediaPlaylist(list []byte) MediaPlaylist {
// 	var mediaList MediaPlaylist
// 	lines := strings.Split(string(list), "\n")

// 	for i := 1; i < len(lines); i++ {
// 		line := lines[i]

// 		parts := strings.Split(line, ":")
// 		if len(parts) < 2 {
// 			continue
// 		}

// 		key := parts[0]
// 		v := parts[1]

// 		switch key {
// 		case "#EXT-X-VERSION":
// 			value, _ := strconv.ParseInt(v, 10, 64)
// 			mediaList.Version = value
// 		case "#EXT-X-TARGETDURATION":
// 			value, _ := strconv.ParseFloat(v, 64)
// 			mediaList.TargetDuration = value
// 		case "#EXT-X-PLAYLIST-TYPE":
// 			mediaList.PlaylistType = v
// 		case "#EXT-X-TWITCH-ELAPSED-SECS":
// 			value, _ := strconv.ParseFloat(v, 64)
// 			mediaList.ElapsedSecs = value
// 		case "#EXT-X-TWITCH-TOTAL-SECS":
// 			value, _ := strconv.ParseFloat(v, 64)
// 			mediaList.TotalSecs = value
// 		case "#ID3-EQUIV-TDTG":
// 			mediaList.Timestamp = v
// 		case "#EXTINF":
// 			// trim ',' from the end
// 			trimmed := v[:len(v)-1]
// 			seconds, _ := strconv.ParseFloat(trimmed, 64)
// 			duration := time.Duration(seconds * float64(time.Second))
// 			seg := Segment{URL: lines[i+1], Duration: duration}
// 			mediaList.Segments = append(mediaList.Segments, seg)
// 			i++
// 		}
// 	}

// 	return mediaList
// }
