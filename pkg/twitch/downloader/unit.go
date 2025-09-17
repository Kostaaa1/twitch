package downloader

import (
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
		return fmt.Sprintf("Unknown(%d)", v)
	}
}

func GetMediaType(s string) MediaType {
	switch s {
	case "clip":
		return TypeClip
	case "video":
		return TypeVOD
	case "stream":
		return TypeLivestream
	default:
		return -1
	}
}

type Unit struct {
	// ID can be: vod id, clip slug or channel name (livestream)
	ID   string
	Type MediaType
	// Quality of media - 1080p60, 720p60, 480p60 ....
	Quality QualityType
	// Used when wanting to download the part of the VOD
	Start time.Duration
	End   time.Duration
	// used as filename
	Title  string
	Writer io.Writer
	Error  error
}

// Used for creating downloadable unit from raw input. Input could either be clip slug, vod id, channel name or url. Based on the input it will detect media type such as livestream, vod, clip. If the input is URL, it will parse the params such as timestamps and those will be represented as Start and End only if those values are not provided in function parameters.Q
func NewUnit(input, quality string, opts ...UnitOption) *Unit {
	unit := &Unit{
		Title: uuid.NewString(),
	}

	unit.ID, unit.Type, unit.Error = parseIDAndMediaType(input)
	if unit.Error != nil {
		return unit
	}

	parsedURL, err := url.Parse(input)

	if err == nil && unit.Type == TypeVOD {
		if unit.Error = parseVodParams(parsedURL, unit); unit.Error != nil {
			return unit
		}
	}

	unit.Quality, unit.Error = getQuality(quality)
	if unit.Error != nil {
		return unit
	}

	for _, opt := range opts {
		opt(unit)
	}

	return unit
}

type UnitOption func(*Unit)

func WithWriter(dir string) UnitOption {
	return func(u *Unit) {
		ext := "mp4"
		if strings.HasPrefix(u.Quality.String(), "audio") {
			ext = "mp3"
		}
		u.Writer, u.Error = fileutil.CreateFile(dir, u.GetTitle(), ext)
	}
}

func WithTimestamps(start, end time.Duration) UnitOption {
	return func(u *Unit) {
		u.Start = start
		u.End = end
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

func (u Unit) GetID() any {
	return u.ID
}

func (u Unit) GetTitle() string {
	return u.ID
}

func getQuality(quality string) (QualityType, error) {
	switch {
	case quality == "" || quality == "best" || strings.HasPrefix(quality, "1080"):
		return Quality1080p60, nil
	case strings.HasPrefix(quality, "720"):
		return Quality720p60, nil
	case strings.HasPrefix(quality, "480"):
		return Quality480p30, nil
	case strings.HasPrefix(quality, "360"):
		return Quality360p30, nil
	case quality == "worst" || strings.HasPrefix(quality, "160"):
		return Quality160p30, nil
	case strings.HasPrefix(quality, "audio"):
		return QualityAudioOnly, nil
	default:
		return 0, fmt.Errorf("invalid quality was provided: %s. valid are: %s", quality, strings.Join(qualities, ", "))
	}
}

func parseIDAndMediaType(input string) (string, MediaType, error) {
	if input == "" {
		return "", 0, errors.New("input cannot be empty")
	}

	if !strings.Contains(input, "http://") && !strings.Contains(input, "https://") {
		if _, parseErr := strconv.ParseInt(input, 10, 64); parseErr == nil {
			return input, TypeVOD, nil
		}
		if len(input) >= 25 {
			return input, TypeClip, nil
		}
		return input, TypeLivestream, nil
	}

	parsedURL, err := url.Parse(input)
	if err != nil {
		return "", 0, err
	}

	if !strings.Contains(parsedURL.Hostname(), "twitch.tv") {
		return "", 0, errors.New("URL must belong to 'twitch.tv'")
	}

	_, id := path.Split(parsedURL.Path)

	switch {
	case strings.Contains(parsedURL.Host, "clips.twitch.tv") || strings.Contains(parsedURL.Path, "/clip/"):
		return id, TypeClip, nil
	case strings.Contains(parsedURL.Path, "/videos/"):
		return id, TypeVOD, nil
	default:
		return id, TypeLivestream, nil
	}
}

func parseVodParams(u *url.URL, unit *Unit) error {
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
