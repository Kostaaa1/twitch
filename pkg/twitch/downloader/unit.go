package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type MediaType int

const (
	TypeClip MediaType = iota
	TypeVOD
	TypeLivestream
)

func (v MediaType) String() string {
	switch v {
	case TypeClip:
		return "clip"
	case TypeVOD:
		return "video"
	case TypeLivestream:
		return "stream"
	default:
		return fmt.Sprintf("Unknown(%d)", v)
	}
}

type Unit struct {
	// ID can be: vod ID, clip slug or channel name (livestream)
	ID      string
	Type    MediaType
	Quality QualityType
	// Used when wanting to download the part of the VOD
	Start  time.Duration
	End    time.Duration
	Title  string
	Writer io.Writer
	Error  error
}

func (u *Unit) FetchTitle(ctx context.Context, c *twitch.Client) {
	switch u.Type {
	case TypeClip:
		clip, err := c.ClipMetadata(ctx, u.ID)
		if err != nil {
			u.Error = err
			return
		}
		u.Title = clip.Title
	case TypeVOD:
		vod, err := c.VideoMetadata(ctx, u.ID)
		if err != nil {
			u.Error = err
			return
		}
		u.Title = vod.Video.Title
	case TypeLivestream:
		stream, err := c.StreamMetadata(ctx, u.ID)
		if err != nil {
			u.Error = err
			return
		}
		u.Title = stream.BroadcastSettings.Title
	}
}

func parseMediaInput(input string) MediaType {
	if _, parseErr := strconv.ParseInt(input, 10, 64); parseErr == nil {
		return TypeVOD
	}

	if len(input) >= 25 {
		return TypeClip
	}

	return TypeLivestream
}

// Used for creating downloadable unit from raw input. Input could either be clip slug, vod id, channel name or url. Based on the input it will detect media type such as livestream, vod, clip. If the input is URL, it will parse the params such as timestamps and those will be represented as Start and End only if those values are not provided in function parameters.Q
func NewUnit(input string, opts ...unitOption) *Unit {
	unit := new(Unit)

	if input == "" {
		unit.Error = errors.New("input is empty")
		return unit
	}

	u, err := url.ParseRequestURI(input)
	if err != nil {
		unit.ID = input
		unit.Type = parseMediaInput(input)
	} else {
		if !strings.Contains(u.Hostname(), "twitch.tv") {
			unit.Error = errors.New("'twitch.tv' missing from the URL")
			return unit
		}

		_, unit.ID = path.Split(u.Path)
		unit.Type = parseMediaInput(unit.ID)

		extractParamsFromURL(u, unit)
	}

	for _, opt := range opts {
		opt(unit)
	}

	return unit
}

type unitOption func(*Unit)

func WithTitle(c *twitch.Client) unitOption {
	return func(u *Unit) {
		u.FetchTitle(context.Background(), c)
	}
}

func WithWriter(dir string) unitOption {
	return func(u *Unit) {
		if u.Error != nil {
			return
		}

		ext := "mp4"
		if strings.HasPrefix(u.Quality.String(), "audio") {
			ext = "mp3"
		}

		u.Writer, u.Error = fileutil.CreateFile(dir, u.GetID(), ext)
	}
}

func WithTimestamps(start, end time.Duration) unitOption {
	return func(u *Unit) {
		u.Start = start
		u.End = end
	}
}

func WithQuality(q string) unitOption {
	return func(u *Unit) {
		switch {
		case q == "" || q == "best" || strings.HasPrefix(q, "1080"):
			u.Quality = Quality1080p60
		case strings.HasPrefix(q, "720"):
			u.Quality = Quality720p60
		case strings.HasPrefix(q, "480"):
			u.Quality = Quality480p30
		case strings.HasPrefix(q, "360"):
			u.Quality = Quality360p30
		case q == "worst" || strings.HasPrefix(q, "160"):
			u.Quality = Quality160p30
		case strings.HasPrefix(q, "audio"):
			u.Quality = QualityAudioOnly
		default:
			u.Quality = 0
			u.Error = fmt.Errorf("invalid quality was provided: %s. valid are: %s", q, strings.Join(qualities, ", "))
		}
	}
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.Writer.(*os.File); ok && f != nil {
		if u.Error != nil {
			os.Remove(f.Name())
		}
		return f.Close()
	}
	return nil
}

// Implement spinner interface
func (u Unit) GetError() error {
	return u.Error
}

func (u Unit) GetID() string {
	if u.Title == "" {
		return u.ID
	}
	return u.Title
}

func extractParamsFromURL(u *url.URL, unit *Unit) error {
	if unit.Start == 0 {
		if t := u.Query().Get("t"); t != "" {
			unit.Start, _ = time.ParseDuration(t)
		}
	}

	if unit.Start > unit.End {
		return fmt.Errorf("invalid time range: start time (%v) must be less than end time (%v) for URL: %s", unit.Start, unit.End, u.String())
	}

	return nil
}
