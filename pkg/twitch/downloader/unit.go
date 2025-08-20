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
	ID string
	// used when creating events
	// UserID string
	// Type can be VOD, Clip, Livestream
	Type MediaType
	// Quality of media - 1080p60, 720p60, 480p60 ....
	Quality QualityType
	// Used when wanting to download the part of the VOD
	Start time.Duration
	End   time.Duration
	// used when building path to writer
	// Title  string
	Writer io.Writer
	Error  error
}

func (u Unit) GetError() error {
	return u.Error
}

func (u Unit) GetTitle() string {
	if f, ok := u.Writer.(*os.File); ok && f != nil {
		return f.Name()
	}
	return u.ID
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

func (unit *Unit) NotifyProgressChannel(msg spinner.ChannelMessage, progressCh chan spinner.ChannelMessage) {
	if progressCh == nil {
		return
	}
	if unit.Writer != nil {
		if file, ok := unit.Writer.(*os.File); ok && file != nil {
			if unit.Error != nil {
				os.Remove(file.Name())
				unit.Writer = nil
			}
			l := msg
			l.Text = file.Name()
			progressCh <- l
		}
	}
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

func getVideoSigAndType(input string) (string, MediaType, error) {
	if input == "" {
		return "", 0, errors.New("input cannot be empty")
	}

	// if its not URL
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

// Used for creating downloadable unit from raw input. Input could either be clip slug, vod id, channel name or url. Based on the input it will detect media type such as livestream, vod, clip. If the input is URL, it will parse the params such as timestamps and those will be represented as Start and End only if those values are not provided in function parameters.Q
func NewUnit(input, quality string, opts ...UnitOption) *Unit {
	unit := &Unit{}

	unit.ID, unit.Type, unit.Error = getVideoSigAndType(input)
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

// func (dl *Downloader) MediaTitle(id string, vtype MediaType) (string, error) {
// 	switch vtype {
// 	case TypeVOD:
// 		data, err := dl.twClient.VideoMetadata(id)
// 		if err != nil {
// 			return "", err
// 		}
// 		return data.Video.Title, nil
// 	case TypeClip:
// 		data, err := dl.twClient.ClipMetadata(id)
// 		if err != nil {
// 			return "", err
// 		}
// 		return data.Video.Title, nil
// 	case TypeLivestream:
// 		data, err := dl.twClient.StreamMetadata(id)
// 		if err != nil {
// 			return "", err
// 		}
// 		return data.BroadcastSettings.Title, nil
// 	default:
// 		return "", errors.New("not found")
// 	}
// }
