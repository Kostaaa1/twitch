package m3u8

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

type Segment struct {
	URL      string
	Duration time.Duration
	Data     chan io.ReadCloser
}

type MediaPlaylist struct {
	URL             string
	Version         int64
	TargetDuration  float64
	Timestamp       string
	PlaylistType    string
	ElapsedSecs     float64
	TotalSecs       float64
	SegmentDuration time.Duration
	Segments        []Segment
}

func (mp *MediaPlaylist) Truncate(start, end time.Duration) {
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

func ParseMediaPlaylist(r io.Reader, url string) (*MediaPlaylist, error) {
	mediaList := &MediaPlaylist{URL: url}
	reader := bufio.NewReader(r)

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to parsed media playlist: %s\n", err.Error())
		}

		id := bytes.IndexByte(line, ':')
		if id == -1 {
			continue
		}

		key := string(line[:id])
		v := string(line[id+1:])

		switch key {
		case "#EXT-X-VERSION":
			value, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			mediaList.Version = value
		case "#EXT-X-TARGETDURATION":
			value, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			mediaList.TargetDuration = value
		case "#EXT-X-PLAYLIST-TYPE":
			mediaList.PlaylistType = v
		case "#EXT-X-TWITCH-ELAPSED-SECS":
			value, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			mediaList.ElapsedSecs = value
		case "#EXT-X-TWITCH-TOTAL-SECS":
			value, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			mediaList.TotalSecs = value
		case "#ID3-EQUIV-TDTG":
			mediaList.Timestamp = v
		case "#EXTINF":
			trimmed := v[:len(v)-1]

			seconds, err := strconv.ParseFloat(trimmed, 64)
			if err != nil {
				return nil, err
			}

			duration := time.Duration(seconds * float64(time.Second))

			segmentURL, _, err := reader.ReadLine()
			if err != nil {
				return nil, fmt.Errorf("failed to read next line: %s", err)
			}

			mediaList.Segments = append(mediaList.Segments, Segment{
				URL:      string(segmentURL),
				Duration: duration,
				Data:     make(chan io.ReadCloser, 1),
			})
		}
	}

	return mediaList, nil
}
