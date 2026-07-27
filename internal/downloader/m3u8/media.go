package m3u8

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Segment struct {
	URI      string
	Duration time.Duration
	Data     chan []byte
}

func (s *Segment) Alloc() { s.Data = make(chan []byte, 1) }

type Map struct {
	URI       string
	ByteRange string
}

type MediaPlaylist struct {
	Source         string
	Version        string
	Timestamp      string
	PlaylistType   string
	TargetDuration string
	ElapsedSecs    string
	TotalSecs      time.Duration
	Map            *Map
	Segments       []*Segment
}

func (mp *MediaPlaylist) Truncate(start, end time.Duration) error {
	if start > mp.TotalSecs {
		return fmt.Errorf("provided start pointer %s exceeds playlist total %s", start, mp.TotalSecs)
	}
	if end < 0 {
		return errors.New("end pointer must not be less then zero")
	}
	if end > 0 && start >= end {
		return errors.New("start pointer must not be equal or larger then end")
	}

	total := time.Duration(0)

	startIdx := 0
	if start > 0 {
		for _, seg := range mp.Segments {
			if total >= start {
				break
			}
			total += seg.Duration
			startIdx++
		}
	}

	endIdx := len(mp.Segments)

	if end > 0 {
		endIdx = startIdx
		for _, seg := range mp.Segments {
			if total >= end {
				break
			}
			total += seg.Duration
			endIdx++
		}
	}

	mp.Segments = slices.Clone(mp.Segments[startIdx:endIdx])

	return nil
}

func (l *MediaPlaylist) parsePlaylistMap(value string) error {
	l.Map = &Map{}
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
			l.Map.URI = value
		case "BYTERANGE":
			l.Map.ByteRange = value
		}
	}

	return nil
}

func (l *MediaPlaylist) parseExtInf(r *bufio.Reader, line string) error {
	trimmed := line[:len(line)-1]

	seconds, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return err
	}

	duration := time.Duration(seconds * float64(time.Second))

	uri, _, err := r.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read next line: %s", err)
	}

	l.Segments = append(l.Segments, &Segment{
		URI:      string(uri),
		Duration: duration,
	})

	return nil
}

func ParseMediaPlaylist(r io.Reader, url string) (*MediaPlaylist, error) {
	l := &MediaPlaylist{Source: url}

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
			l.Version = value
		case "#EXT-X-MAP":
			if err := l.parsePlaylistMap(value); err != nil {
				return nil, err
			}
		case "#EXT-X-TARGETDURATION":
			l.TargetDuration = value
		case "#EXT-X-PLAYLIST-TYPE":
			l.PlaylistType = value
		case "#EXT-X-TWITCH-ELAPSED-SECS":
			l.ElapsedSecs = value
		case "#EXT-X-TWITCH-TOTAL-SECS":
			total, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, err
			}
			l.TotalSecs = time.Duration(total * float64(time.Second))
		case "#ID3-EQUIV-TDTG":
			l.Timestamp = value
		case "#EXTINF":
			if err := l.parseExtInf(reader, value); err != nil {
				return nil, err
			}
		}
	}

	return l, nil
}
