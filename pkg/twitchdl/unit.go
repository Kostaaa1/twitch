package twitchdl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
)

type VideoType int

const (
	TypeClip VideoType = iota
	TypeVOD
	TypeLivestream
)

type MediaUnit struct {
	ID      string
	Type    VideoType
	Quality string
	Start   time.Duration
	End     time.Duration
	Writer  io.Writer
	Error   error
}

func (mu MediaUnit) GetError() error {
	return mu.Error
}

func (mu MediaUnit) GetTitle() string {
	if f, ok := mu.Writer.(*os.File); ok && f != nil {
		if mu.Error != nil {
			os.Remove(f.Name())
		}
		return f.Name()
	}
	return mu.ID
}

// refactor this
func NewMediaUnit(URL, quality, output string, start, end time.Duration) MediaUnit {
	var unit MediaUnit

	parsed, err := ParseURL(URL)
	if err != nil {
		unit.Error = err
	}

	if parsed.Type == TypeVOD {
		if start > 0 && end > 0 && start >= end {
			unit.Error = fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL: %s", start, end, URL)
		}
	}

	quality, err = GetQuality(quality, parsed.Type)
	if err != nil {
		unit.Error = err
	}

	mediaName := fmt.Sprintf("%s_%s", parsed.ID, quality)
	ext := "mp4"
	if quality == "audio_only" {
		ext = "mp3"
	}

	// can i avoid creating if error occurs? also, should i close the file?
	var f *os.File
	if unit.Error == nil {
		f, err = fileutil.CreateFile(output, mediaName, ext)
		if err != nil {
			unit.Error = err
		}
	}

	unit.ID = parsed.ID
	unit.Type = parsed.Type
	unit.Quality = quality
	if start > 0 {
		unit.Start = start
	} else {
		unit.Start = parsed.TrimStart
	}
	unit.End = end
	unit.Writer = f

	return unit
}

type ParsedURL struct {
	ID        string
	Type      VideoType
	TrimStart time.Duration
}

func ParseURL(URL string) (*ParsedURL, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the URL: %s", err)
	}

	if !strings.Contains(u.Hostname(), "twitch.tv") {
		return nil, fmt.Errorf("the hostname of the URL does not contain twitch.tv")
	}

	if strings.Contains(u.Host, "clips.twitch.tv") || strings.Contains(u.Path, "/clip/") {
		_, id := path.Split(u.Path)
		return &ParsedURL{
			ID:   id,
			Type: TypeClip,
		}, nil
	}

	if strings.Contains(u.Path, "/videos/") {
		t := u.Query().Get("t")

		start, err := time.ParseDuration(t)
		if err != nil {
			return nil, errors.New("timestamp not valid format. valid - [1h3m22s]")
		}

		_, id := path.Split(u.Path)

		return &ParsedURL{
			ID:        id,
			Type:      TypeVOD,
			TrimStart: start,
		}, nil
	}

	s := strings.Split(u.Path, "/")
	return &ParsedURL{ID: s[1], Type: TypeLivestream}, nil
}

func extractAudio(segmentURL string, w io.Writer) (int64, error) {
	cmd := exec.Command("ffmpeg", "-i", segmentURL, "-q:a", "0", "-map", "a", "-f", "mp3", "-")
	cmd.Stdout = nil
	cmd.Stderr = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	n, err := io.Copy(w, stdout)
	if err != nil {
		return 0, fmt.Errorf("failed to copy audio data: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return 0, fmt.Errorf("FFmpeg conversion failed: %w", err)
	}

	return n, nil
}
