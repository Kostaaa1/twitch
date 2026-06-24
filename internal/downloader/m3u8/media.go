package m3u8

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Segment struct {
	URI      string
	Duration time.Duration
	Data     chan io.ReadCloser
}

type Map struct {
	URI       string
	ByteRange string
}

type MediaPlaylist struct {
	URL             string
	Version         string
	Timestamp       string
	PlaylistType    string
	TargetDuration  string
	ElapsedSecs     string
	TotalSecs       string
	Map             *Map
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

func parsePlaylistMap(list *MediaPlaylist, value string) error {
	list.Map = &Map{}
	values := strings.Split(value, ",")

	for _, value := range values {
		parts := strings.Split(value, "=")

		if len(parts) != 2 {
			return errors.New("malformed playlist")
		}

		value, err := strconv.Unquote(parts[1])
		if err != nil {
			value = parts[1]
		}

		switch parts[0] {
		case "URI":
			list.Map.URI = value
		case "BYTERANGE":
			list.Map.ByteRange = value
		}
	}

	return nil
}

func parseExtInf(r *bufio.Reader, list *MediaPlaylist, line string) error {
	trimmed := line[:len(line)-1]
	seconds, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return err
	}

	duration := time.Duration(seconds * float64(time.Second))

	segmentURL, _, err := r.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read next line: %s", err)
	}

	list.Segments = append(list.Segments, Segment{
		URI:      string(segmentURL),
		Duration: duration,
		Data:     make(chan io.ReadCloser, 1),
	})

	return nil
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
		value := string(line[id+1:])

		switch key {
		case "#EXT-X-VERSION":
			mediaList.Version = value
		case "#EXT-X-MAP":
			parsePlaylistMap(mediaList, value)
		case "#EXT-X-TARGETDURATION":
			mediaList.TargetDuration = value
		case "#EXT-X-PLAYLIST-TYPE":
			mediaList.PlaylistType = value
		case "#EXT-X-TWITCH-ELAPSED-SECS":
			mediaList.ElapsedSecs = value
		case "#EXT-X-TWITCH-TOTAL-SECS":
			mediaList.TotalSecs = value
		case "#ID3-EQUIV-TDTG":
			mediaList.Timestamp = value
		case "#EXTINF":
			parseExtInf(reader, mediaList, value)
		}
	}

	return mediaList, nil
}
