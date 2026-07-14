package downloader

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	UUID       uuid.UUID
	ID         string
	Type       MediaType
	Quality    QualityType
	Start, End time.Duration
	Error      error

	mu                 sync.Mutex
	w                  io.Writer
	dir, filename, ext string
	title              string
	audioRecoverable   bool
}

func (u *Unit) getAudioRecoverable() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.audioRecoverable
}

func (u *Unit) setAudiorecoverable(v bool) {
	u.mu.Lock()
	u.audioRecoverable = v
	u.mu.Unlock()
}

func (u *Unit) ensureExt(url string) {
	u.mu.Lock()
	if u.ext == "" {
		paramID := strings.LastIndex(url, "?")
		if paramID != -1 {
			u.ext = filepath.Ext(url[:paramID])
		} else {
			u.ext = filepath.Ext(url)
		}
	}
	u.mu.Unlock()
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

func NewUnit(input string, opts ...unitOption) *Unit {
	unit := &Unit{
		UUID:             uuid.New(),
		audioRecoverable: true,
	}

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

func (u *Unit) restampFrames() error {
	// when downloading vod, it can happen that interval between frames is not steady, for example if source vod should be at 60 FPS, meaning that each frame needs to be displayed at 16.667 ms rate interval, but this isn't the case sometimes. some frames are at rounded 16/17 ms, which will cause occasional video playback jitter. we can fix this and snap them at this fixed interval. basically scheduling issue (no touching frames, just rescheduling)
	if f, ok := u.w.(*os.File); ok {
		fname := f.Name()
		cmd := exec.Command(
			"ffmpeg", "-y",
			"-i", fname,
			"-c", "copy",
			"-bsf:v", "setts=pts=round((PTS-STARTPTS)/1500)*1500+STARTPTS:dts=round((DTS-STARTDTS)/1500)*1500+STARTDTS",
			"-movflags", "+faststart",
			strings.Replace(fname, filepath.Ext(f.Name()), ".mp4", 1),
		)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.w.(*os.File); ok && f != nil {
		return f.Close()
	}
	return nil
}

func (u *Unit) GetLabel() string {
	if u.title != "" {
		return u.title
	}
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
