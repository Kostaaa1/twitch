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

type metadata struct {
	title       string
	date        time.Time
	id          string
	channelName string
	clipCreator *string
}

type Unit struct {
	UUID       uuid.UUID
	ID         string
	Type       MediaType
	Quality    QualityType
	Start, End time.Duration
	Error      error

	w                  io.Writer
	dir, filename, ext string
	total              float64
	template           string
	metadata           *metadata
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

func WithPathname(pathname string) unitOption {
	return func(u *Unit) error {
		if pathname == "" {
			return errors.New("failed to parse provided pathname: pathanme is not provided")
		}

		stat, err := os.Stat(pathname)
		if os.IsNotExist(err) {
			dir, file := filepath.Split(pathname)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return fmt.Errorf("failed to parse provided pathname: provided pathname is not valid: %s", pathname)
			}
			u.dir = dir
			u.filename = file
			return nil
		}

		if err != nil {
			return err
		}

		if stat.IsDir() {
			u.filename = ""
			u.dir = pathname
			return nil
		}

		dir, file := filepath.Split(pathname)
		u.dir = dir
		u.filename = file

		return nil
	}
}

func WIthTemplate(template string) unitOption {
	return func(u *Unit) error {
		u.template = template
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

func (u *Unit) parseTemplate() string {
	bid := -1
	var s strings.Builder

	for id, r := range u.template {
		switch r {
		case '{':
			if bid == -1 {
				bid = id
			}
		case '}':
			if bid == -1 {
				continue
			}

			key := u.template[bid:id]

			switch key {
			case "channel":
				s.WriteString(u.metadata.channelName)
			case "date":
				s.WriteString(u.metadata.date.String())
			case "quality":
				s.WriteString(u.Quality.String())
			case "name":
				title := u.metadata.title
				if len(title) > 60 {
					s.WriteString(u.metadata.title[:60])
				} else {
					s.WriteString(u.metadata.title)
				}
			case "id":
				s.WriteString(u.metadata.id)
			}

			bid = -1
		default:
			s.WriteRune(r)
		}
	}

	return s.String()
}

// func (u *Unit) OpenFile(ctx context.Context, dl *Downloader) error {
// 	if u.w != nil {
// 		return nil
// 	}

// 	if u.dir == "" {
// 		return errors.New("missing dir")
// 	}
// 	if u.ext == "" {
// 		return errors.New("missing file extension")
// 	}

// 	if u.filename == "" {
// 		// title, err := dl.fetchTitle(ctx, u)
// 		// if err != nil {
// 		// 	return err
// 		// }
// 		// u.filename = title
// 	}

// 	pathname, err := fileutil.ConstructPathname(u.dir, u.filename, u.ext)
// 	if err != nil {
// 		return err
// 	}

// 	f, err := os.OpenFile(pathname, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
// 	if err != nil {
// 		return err
// 	}
// 	u.w = f

// 	return nil
// }

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
