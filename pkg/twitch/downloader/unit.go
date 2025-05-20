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

	"github.com/Kostaaa1/twitch/pkg/spinner"
)

type VideoType int

const (
	TypeClip VideoType = iota
	TypeVOD
	TypeLivestream
)

type UnitOption func(*Unit)

func WithWriter(w io.Writer) UnitOption {
	return func(u *Unit) {
		u.Writer = w
	}
}

func WithTimestamps(start, end time.Duration) UnitOption {
	return func(u *Unit) {
		u.Start = start
		u.End = end
	}
}

type Unit struct {
	// Vod id, clip slug or channel name
	ID      string
	Type    VideoType
	Quality QualityType
	Start   time.Duration
	End     time.Duration
	Writer  io.Writer
	Error   error
}

func (v VideoType) String() string {
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

func GetVideoType(s string) VideoType {
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

// needed for spinner interface
func (mu Unit) GetError() error {
	return mu.Error
}

func (mu Unit) GetTitle() string {
	if f, ok := mu.Writer.(*os.File); ok && f != nil {
		// if mu.Error != nil {
		// 	os.Remove(f.Name())
		// }
		return f.Name()
	}
	return mu.ID
}

func (unit Unit) NotifyProgressChannel(msg spinner.ChannelMessage, progressCh chan spinner.ChannelMessage) {
	if progressCh == nil {
		return
	}
	fmt.Println("notifying channel")
	if unit.Writer != nil {
		if file, ok := unit.Writer.(*os.File); ok && file != nil {
			// if unit.Error != nil {
			// 	os.Remove(file.Name())
			// }
			l := msg
			l.Text = file.Name()
			progressCh <- l
		}
	}
}

func qualityFromInput(quality string) (QualityType, error) {
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

func parseVideoType(input string) (string, VideoType, error) {
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

// Used for creating downloadable unit from raw input. Input could either be clip slug, vod id, channel name or url. Based on the input it will detect media type such as livestream, vod, clip. If the input is URL, it will parse the params such as timestamps and those will be represented as Start and End only if those values are not provided in function parameters.
func NewUnit(input, quality string, opts ...UnitOption) *Unit {
	unit := &Unit{}

	unit.ID, unit.Type, unit.Error = parseVideoType(input)
	if unit.Error != nil {
		return unit
	}

	if unit.Type == TypeVOD {
		if unit.Error = parseVodParams(input, unit); unit.Error != nil {
			return unit
		}
	}

	unit.Quality, unit.Error = qualityFromInput(quality)
	if unit.Error != nil {
		return unit
	}

	for _, opt := range opts {
		opt(unit)
	}

	return unit
}

func parseVodParams(input string, unit *Unit) error {
	parsedURL, err := url.Parse(input)
	if err != nil {
		return err
	}
	if unit.Start == 0 {
		if t := parsedURL.Query().Get("t"); t != "" {
			unit.Start, _ = time.ParseDuration(t)
		}
	}
	if unit.Start > unit.End {
		return fmt.Errorf("invalid time range: start time (%v) must be less than end time (%v) for URL: %s", unit.Start, unit.End, input)
	}
	return nil
}

func (dl *Downloader) MediaTitle(id string, vtype VideoType) (string, error) {
	switch vtype {
	case TypeVOD:
		data, err := dl.TWApi.VideoMetadata(id)
		if err != nil {
			return "", err
		}
		return data.Video.Title, nil
	case TypeClip:
		data, err := dl.TWApi.ClipMetadata(id)
		if err != nil {
			return "", err
		}
		return data.Video.Title, nil
	case TypeLivestream:
		data, err := dl.TWApi.StreamMetadata(id)
		if err != nil {
			return "", err
		}
		return data.BroadcastSettings.Title, nil
	default:
		return "", errors.New("not found")
	}
}
