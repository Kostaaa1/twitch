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
	w                  io.Writer
	dir, filename, ext string
	Title              string
}

func (u *Unit) Validate() error {
	if u.Type < 0 || u.Type > 2 {
		return errors.New("unit type is not valid")
	}
	return nil
}

type unitOption func(*Unit) error

func WithPathname(pathname string) unitOption {
	return func(u *Unit) error {
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
		u.filename = filepath.Base(pathname)

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

func NewUnit(input string, opts ...unitOption) (*Unit, error) {
	if input == "" {
		return nil, errors.New("missing input: please provide input (clip slug | vod id | channel name to record livestream)")
	}

	unit := &Unit{UUID: uuid.New()}

	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		unit.ID = input
	} else {
		unit.parseTwitchURL(parsedURL)
	}

	unit.Type = discoverUnitType(unit.ID)

	for _, opt := range opts {
		if err := opt(unit); err != nil {
			return nil, err
		}
	}

	if unit.w == nil && unit.dir == "" {
		return nil, errors.New("missing writer or pathname: must provide either")
	}

	return unit, nil
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.w.(*os.File); ok && f != nil {
		return f.Close()
	}
	return nil
}

// spinner interface
func (u Unit) GetLabel() string {
	return u.Title
}

func (u Unit) GetID() string {
	return u.UUID.String()
}
