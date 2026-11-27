package downloader

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/google/uuid"
)

const (
	maxUnmuteFailStreak int32 = 40
)

type Metadata struct {
	ID      string // VOD id or clip slug
	Channel string // channel name
	Quality string // the quality actually downloaded, e.g. "1080p60" (not "best")
	Name    string // title
	Clipper string // clips only
	Date    time.Time
}

type Unit struct {
	UUID       uuid.UUID
	ID         string
	Type       MediaType
	Quality    QualityType
	Start, End time.Duration
	Error      error

	input              string
	w                  io.Writer
	dir, filename, ext string
	total              float64

	metadata         *Metadata
	unmuteFailStreak atomic.Int32
	notifyFn         func(Progress)
}

// TODO: revisit which params can be parsed
func (u *Unit) parseTwitchURL(url *url.URL) error {
	if !strings.Contains(url.Hostname(), "twitch.tv") {
		return errors.New("'twitch.tv' missing from the URL")
	}

	u.ID = path.Base(url.Path)
	if u.Start == 0 {
		if t := url.Query().Get("t"); t != "" {
			s, err := time.ParseDuration(t)
			if err != nil {
				return err
			}
			u.Start = s
		}
	}

	return nil
}

func discoverUnitType(input string) MediaType {
	if _, parseErr := strconv.ParseInt(input, 10, 64); parseErr == nil {
		return TypeVOD
	}
	if len(input) > 25 {
		return TypeClip
	}
	return TypeLivestream
}

var templateFields = map[string]string{
	"{channel}": "",
	"{name}":    "",
	"{id}":      "",
	"{quality}": "",
	"{date}":    "",
}

// TODO::::::
// "video":  "{date}_{name}_{id}",
// "clip":   "{channel}/clips/{date}_{name}_{id}",
// "stream": "{channel}/live/{date}_{time}"
func (u *Unit) ParseTemplate(template string) (string, error) {
	_, path := filepath.Split(template)
	if len(path) == 0 {
		return "", errors.New("")
	}

	var s strings.Builder
	pathparts := strings.Split(path, "_")
	for id, p := range pathparts {
		if _, ok := templateFields[p]; !ok {
			continue
		}

		var v string

		switch p {
		case "channel":
		case "name":
		case "id":
		case "quality":
		case "date":
		}

		if v == "" {
			panic("assert: missing value")
		}

		if id != len(pathparts)-1 {
			v += "_"
		}

		s.WriteString(v)
	}

	return "", nil
}

func (u *Unit) setFileExt(segURL string) error {
	if u.w != nil || u.ext != "" {
		return nil
	}
	// ext must be discovered by checking playlist
	// if u.ext != "" {
	// 	return errors.New("ext already provided")
	// }
	paramID := strings.LastIndex(segURL, "?")
	if paramID != -1 {
		u.ext = filepath.Ext(segURL[:paramID])
	} else {
		u.ext = filepath.Ext(segURL)
	}
	return nil
}

func (u *Unit) SetProgressNotifier(fn func(Progress)) { u.notifyFn = fn }

func (u *Unit) CloseWriter() error {
	if f, ok := u.w.(io.WriteCloser); ok && f != nil {
		return f.Close()
	}
	return nil
}

type UnitOption func(*Unit) error

func WithNotifyFn(fn func(Progress)) UnitOption {
	return func(u *Unit) error {
		u.notifyFn = fn
		return nil
	}
}

func WithWriter(w io.Writer) UnitOption {
	return func(u *Unit) error {
		u.w = w
		return nil
	}
}

func WithPath(dir string, forceDir bool) UnitOption {
	return func(u *Unit) error {
		if err := fileutil.ParsePath(dir, forceDir); err != nil {
			return err
		}
		u.dir = dir
		return nil
	}
}

func WithTimestamps(start, end time.Duration) UnitOption {
	return func(u *Unit) error {
		if u.Type != TypeVOD {
			return nil
		}
		if start != 0 {
			u.Start = start
		}
		if end != 0 {
			u.End = end
		}
		return nil
	}
}

func WithQuality(q string) UnitOption {
	return func(u *Unit) error {
		u.Quality, u.Error = ParseQuality(q)
		return u.Error
	}
}

func ValidateStartEnd(start, end time.Duration, totalSec *int) error {
	if end < 0 {
		return fmt.Errorf("")
	}
	if start < 0 {
		return fmt.Errorf("")
	}
	if end != 0 && start >= end {
		return fmt.Errorf("invalid time range: start (%v) must be before end (%v)", start, end)
	}
	return nil
}

func (u *Unit) Validate() error {
	if err := ValidateStartEnd(u.Start, u.End, nil); err != nil {
		return err
	}
	if u.w == nil && u.dir == "" {
		return errors.New("missing writer or pathname: must provide either")
	}
	return nil
}

func NewUnit(input string, opts ...UnitOption) *Unit {
	u := &Unit{
		UUID:  uuid.New(),
		input: input,
	}

	if input == "" {
		u.Error = errors.New("missing input: please provide input (clip slug | vod id | channel name to record livestream)")
		return u
	}

	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		u.ID = input
	} else {
		u.Error = u.parseTwitchURL(parsedURL)
		if u.Error != nil {
			return u
		}
	}

	u.Type = discoverUnitType(u.ID)

	for _, opt := range opts {
		if err := opt(u); err != nil {
			u.Error = err
			return u
		}
	}

	u.Validate()

	return u
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
