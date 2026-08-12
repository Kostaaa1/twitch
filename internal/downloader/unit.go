package downloader

import (
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

	"github.com/google/uuid"
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
		return fmt.Sprintf("Unknown (%d)", v)
	}
}

type Unit struct {
	UUID               uuid.UUID
	ID                 string
	Type               MediaType
	Quality            QualityType
	Start, End         time.Duration
	Error              error
	w                  io.Writer
	dir, filename, ext string
	total              float64
}

func (u *Unit) setFileExt(url string) error {
	if u.w != nil {
		return nil
	}

	// ext must be discovered by checking playlist
	if u.ext != "" {
		return errors.New("ext already provided")
	}

	paramID := strings.LastIndex(url, "?")
	if paramID != -1 {
		u.ext = filepath.Ext(url[:paramID])
	} else {
		u.ext = filepath.Ext(url)
	}

	return nil
}

func (u *Unit) Validate() error {
	if u.Type < 0 || u.Type > 2 {
		return errors.New("unit type is not valid")
	}
	return nil
}

type unitOption func(*Unit) error

func WithWriter(w io.Writer) unitOption {
	return func(u *Unit) error {
		u.w = w
		return nil
	}
}

// parses provided pathname, validates dir and sets the dir and filename (if provided) fields. provided extension will be discarded as it needs to be discovered when playlist is fetched (as m3u8 playlist segments can be of multiple formats)
func WithPathname(pathname string) unitOption {
	return func(u *Unit) error {
		if pathname == "" {
			pathname = "./"
		}

		info, err := os.Stat(pathname)
		if err == nil && info.IsDir() {
			u.dir = pathname
			u.filename = ""
			return nil
		}

		dir := filepath.Dir(pathname)
		if _, err := os.Stat(dir); err != nil {
			return err
		}

		u.dir = dir
		base := filepath.Base(pathname)
		u.filename = strings.TrimSuffix(base, filepath.Ext(base))

		return nil
	}
}

// sets start/end for VOD units
func WithTimestamps(start, end time.Duration) unitOption {
	return func(u *Unit) error {
		if u.Type != TypeVOD {
			return nil
		}
		u.Start = start
		u.End = end
		return nil
	}
}

// sets the quality, throws an error if provided quality is not valid
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

func (u *Unit) parseTwitchURL(url *url.URL) error {
	if !strings.Contains(url.Hostname(), "twitch.tv") {
		return errors.New("'twitch.tv' missing from the URL")
	}

	_, u.ID = path.Split(url.Path)
	if u.Start == 0 {
		if t := url.Query().Get("t"); t != "" {
			s, err := time.ParseDuration(t)
			if err != nil {
				return err
			}
			u.Start = s
		}
	}

	if u.Start > u.End {
		return fmt.Errorf("invalid time range: start time (%v) must be less than end time (%v) for URL: %s", u.Start, u.End, url.String())
	}

	return nil
}

func discoverUnitType(input string) MediaType {
	if _, parseErr := strconv.ParseInt(input, 10, 64); parseErr == nil {
		return TypeVOD
	}
	if len(input) >= 25 {
		return TypeClip
	}
	return TypeLivestream
}

func NewUnit(input string, opts ...unitOption) *Unit {
	unit := &Unit{UUID: uuid.New()}

	if input == "" {
		unit.Error = errors.New("missing input: please provide input (clip slug | vod id | channel name to record livestream)")
		return unit
	}

	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		unit.ID = input
	} else {
		unit.parseTwitchURL(parsedURL)
	}

	unit.Type = discoverUnitType(unit.ID)

	for _, opt := range opts {
		if err := opt(unit); err != nil {
			unit.Error = err
			return unit
		}
	}

	if unit.w == nil && unit.dir == "" {
		unit.Error = errors.New("missing writer or pathname: must provide either")
	}

	return unit
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.w.(io.WriteCloser); ok && f != nil {
		return f.Close()
	}
	return nil
}

// implement progress spinner interface
func (u *Unit) GetLabel() string {
	if u.filename != "" {
		return u.filename
	}
	return u.ID
}

func (u *Unit) GetID() string {
	return u.UUID.String()
}

func (u *Unit) GetError() error {
	return u.Error
}
