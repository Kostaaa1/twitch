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

func parseMediaInput(input string) MediaType {
	if _, parseErr := strconv.ParseInt(input, 10, 64); parseErr == nil {
		return TypeVOD
	}
	if len(input) >= 25 {
		return TypeClip
	}
	return TypeLivestream
}

type Unit struct {
	// id of the vod, slug of the clip, channel name (for stream)
	ID string
	// type of media: vod, clip, stream, highlight
	Type MediaType
	// type of quality:
	Quality QualityType
	// timestamps
	Start, End time.Duration
	// title of the media - WithTitle fetches the title based on the mediatype
	Title string
	// error
	Error error
	// writer
	Writer io.Writer
	// pathname for file writer - file writer gets created upon write
	pathname string
}

func (u *Unit) fetchTitle(ctx context.Context, c *twitch.Client) error {
	switch u.Type {
	case TypeClip:
		title, err := c.Gql.ClipTitle(ctx, u.ID)
		if err != nil {
			u.Error = err
			return err
		}
		u.Title = title
	case TypeVOD:
		title, err := c.Gql.VideoTitle(ctx, u.ID)
		if err != nil {
			u.Error = err
			return err
		}
		u.Title = title
	case TypeLivestream:
		title, err := c.Gql.StreamTitle(ctx, u.ID)
		if err != nil {
			u.Error = err
			return err
		}
		u.Title = title
	}

	return nil
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

func NewUnit(input string, opts ...unitOption) (*Unit, error) {
	unit := new(Unit)

	if input == "" {
		unit.Error = errors.New("input is empty")
		return unit, unit.Error
	}

	u, err := url.ParseRequestURI(input)
	if err != nil {
		unit.ID = input
	} else {
		if !strings.Contains(u.Hostname(), "twitch.tv") {
			unit.Error = errors.New("'twitch.tv' missing from the URL")
			return unit, unit.Error
		}
		_, unit.ID = path.Split(u.Path)
		extractParamsFromURL(u, unit)
	}

	unit.Type = parseMediaInput(unit.ID)

	for _, opt := range opts {
		opt(unit)
	}

	if unit.Writer == nil {
		if unit.pathname == "" {
			unit.Error = errors.New("")
		}
	}

	return unit, nil
}

type unitOption func(*Unit)

func WithTitle(c *twitch.Client) unitOption {
	return func(u *Unit) {
		u.fetchTitle(context.Background(), c)
	}
}

func WithWriter(w io.WriteCloser) unitOption {
	return func(u *Unit) {
		u.Writer = w
	}
}

func WithFile(ctx context.Context, c *twitch.Client, dir string) unitOption {
	return func(u *Unit) {
		if u.Error != nil {
			return
		}

		var ext string
		if u.Type == TypeLivestream || u.Type == TypeVOD {
			ext = "ts"
		}
		if strings.HasPrefix(u.Quality.String(), "audio") {
			ext = "mp3"
		}
		if u.Type == TypeClip {
			ext = "mp4"
		}

		if ext == "" {
			u.Error = errors.New("couldn't extract the file extension")
			return
		}

		u.fetchTitle(ctx, c)

		if u.Title != "" {
			pathname, err := fileutil.ConstructPathname(dir, u.Title, ext)
			if err != nil {
				u.Error = err
				return
			}
			u.pathname = pathname
		}
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
