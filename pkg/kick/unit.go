package kick

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/google/uuid"
)

type Unit struct {
	UUID    uuid.UUID
	Channel string
	Quality string
	Start   time.Duration
	End     time.Duration
	Error   error
	Title   string
	W       io.Writer
}

type unitOptions func(*Unit)

func WithPathname(pathname string) unitOptions {
	return func(u *Unit) {
		if pathname == "" {
			pathname = "./"
		}

		var dir, filename, ext string

		info, err := os.Stat(pathname)
		if err == nil && info.IsDir() {
			dir = pathname
			filename = ""
		} else {
			dir = filepath.Dir(pathname)
			if _, err := os.Stat(dir); err != nil {
				u.Error = err
				return
			}
			base := filepath.Base(pathname)
			filename = strings.TrimSuffix(base, filepath.Ext(base))
		}

		ext = "mp4"
		newp, err := fileutil.ConstructPathname(dir, filename, ext)
		if err != nil {
			u.Error = err
			return
		}

		u.W, u.Error = os.OpenFile(newp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}
}

func WithTimestamps(start, end time.Duration) unitOptions {
	return func(u *Unit) {
		u.Start = start
		u.End = end
	}
}

func NewUnit(videoURL, quality string, opts ...unitOptions) *Unit {
	unit := new(Unit)

	if err := validateQuality(quality); err != nil {
		unit.Error = err
		return unit
	}

	parsed, err := url.Parse(videoURL)
	if err != nil {
		fmt.Println("FAILED TO PARSE?", parsed, videoURL)
		unit.Error = err
		return unit
	}

	pathParts := strings.Split(parsed.Path, "/")
	channel := pathParts[1]
	videoID := pathParts[len(pathParts)-1]

	id, err := uuid.Parse(videoID)
	if err != nil {
		unit.Error = err
		return unit
	}

	unit.Channel = channel
	unit.UUID = id

	for _, opt := range opts {
		opt(unit)
	}

	return unit
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.W.(*os.File); ok && f != nil {
		if u.Error != nil {
			os.Remove(f.Name())
		}
		return f.Close()
	}
	return nil
}

func validateQuality(q string) error {
	valid := []string{"best", "1080", "720", "480", "360", "160", "worst"}
	for _, v := range valid {
		if strings.HasPrefix(v, q) {
			return nil
		}
	}
	return fmt.Errorf("error: invalid quality")
}

// implement progress spinner interface
func (u *Unit) GetLabel() string {
	return u.UUID.String()
}

func (u *Unit) GetID() string {
	return u.UUID.String()
}

func (u *Unit) GetError() error {
	return u.Error
}
