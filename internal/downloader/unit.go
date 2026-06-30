package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
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
	// id of the vod, slug of the clip, channel name (for stream)
	ID string
	// type of media: vod, clip, stream, highlight
	Type MediaType
	// type of quality:
	Quality QualityType
	// timestamps
	Start, End time.Duration
	// writer
	Writer io.Writer
	// Error
	Err      error
	pathname string
}

func (u *Unit) Validate() error {
	// Validate type
	if u.Type < 0 || u.Type > 2 {
		return errors.New("unit type is not valid")
	}
	// Validate quality
	return nil
}

type unitOption func(*Unit) error

func WithWriter(w io.WriteCloser) unitOption {
	return func(u *Unit) error {
		u.Writer = w
		return nil
	}
}

func WithFile(ctx context.Context, c *twitch.Client, pathname string) unitOption {
	return func(u *Unit) error {
		// detect file extension based on twitch media type (vidoe/clip/stream)
		// this wont work as m3u8 playlists can have ts/mp4 (maybe more) segments.
		// to know the file extension, we need to fetch the playlist and inspect it

		if u.Type < 0 || u.Type > 2 {
			return errors.New("invalid unit type")
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
			return errors.New("couldn't extract the file extension")
		}

		dir, filename := filepath.Split(pathname)
		if filepath.Ext(filename) == "" || filename != "" {
			title, err := u.fetchTitle(ctx, c)
			if err != nil {
				return err
			}
			dir = pathname
			filename = title
		}

		pathname, err := fileutil.ConstructPathname(dir, filename, ext)

		u.pathname = pathname

		if err != nil {
			return err
		}

		f, err := os.OpenFile(pathname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		u.Writer = f

		return nil
	}
}

func WithTimestamps(start, end time.Duration) unitOption {
	return func(u *Unit) error {
		u.Start = start
		u.End = end
		return nil
	}
}

func WithQuality(q string) unitOption {
	return func(u *Unit) error {
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
			return fmt.Errorf("invalid quality was provided: %s. valid are: %s", q, strings.Join(qualities, ", "))
		}
		return nil
	}
}

func (u *Unit) fetchTitle(ctx context.Context, c *twitch.Client) (title string, err error) {
	switch u.Type {
	case TypeClip:
		title, err = c.Gql.ClipTitle(ctx, u.ID)
		return
	case TypeVOD:
		title, err = c.Gql.VideoTitle(ctx, u.ID)
		return
	case TypeLivestream:
		title, err = c.Gql.StreamTitle(ctx, u.ID)
		return
	}
	return
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

func mediaTypeFromInput(input string) MediaType {
	if _, parseErr := strconv.ParseInt(input, 10, 64); parseErr == nil {
		return TypeVOD
	}
	if len(input) >= 25 {
		return TypeClip
	}
	return TypeLivestream
}

func NewUnit(input string, opts ...unitOption) *Unit {
	unit := new(Unit)

	if input == "" {
		unit.Err = errors.New("missing input: please provide input (clip slug | vod id | channel name to record livestream)")
		return unit
	}

	u, err := url.ParseRequestURI(input)
	if err != nil {
		unit.ID = input
	} else {
		if !strings.Contains(u.Hostname(), "twitch.tv") {
			unit.Err = errors.New("'twitch.tv' missing from the URL")
			return unit
		}
		_, unit.ID = path.Split(u.Path)
		extractParamsFromURL(u, unit)
	}

	unit.Type = mediaTypeFromInput(unit.ID)

	for _, opt := range opts {
		if err := opt(unit); err != nil {
			unit.Err = err
			return unit
		}
	}

	if unit.Writer == nil && unit.pathname == "" {
		unit.Err = errors.New("missing writer or pathname: must provider either")
		return unit
	}

	return unit
}

func (u *Unit) download(dl *Downloader, r io.ReadCloser) error {
	defer r.Close()

	if u.Writer == nil {
		if u.pathname == "" {
			return errors.New("missing output pathname")
		}
	}

	n, err := io.Copy(u.Writer, r)
	if err != nil {
		return err
	}

	dl.notify(Progress{ID: u.GetID(), Bytes: n})

	return nil
}

func (u *Unit) segmentFetchDownload(ctx context.Context, dl *Downloader, segURL string) error {
	body, err := dl.fetchSegment(ctx, segURL)
	if err != nil {
		return err
	}
	return u.download(dl, body)
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.Writer.(*os.File); ok && f != nil {
		return f.Close()
	}
	return nil
}

// this needs to return unique id
func (u Unit) GetID() string {
	if u.pathname != "" {
		return u.pathname
	}
	if u.ID != "" {
		return u.ID
	}
	return "Unknown (error)"
}
