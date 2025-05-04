package twitchdl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
)

type VideoType int

const (
	TypeClip VideoType = iota
	TypeVOD
	TypeLivestream
)

type Unit struct {
	// unique identifier. vod ID or clip Slug
	ID      string
	URL     string
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

func (mu Unit) GetError() error {
	return mu.Error
}

func (mu Unit) GetTitle() string {
	if f, ok := mu.Writer.(*os.File); ok && f != nil {
		if mu.Error != nil { // ??
			os.Remove(f.Name())
		}
		return f.Name()
	}
	return mu.ID
}

func (dl *Downloader) NewUnit(URL, quality, output string, start, end time.Duration) Unit {
	du := Unit{
		Start: start,
		End:   end,
	}

	parsedURL, err := url.Parse(URL)
	if err != nil {
		du.Error = err
		return du
	}

	if !strings.Contains(parsedURL.Hostname(), "twitch.tv") {
		du.Error = errors.New("the hostname of the URL does not contain twitch.tv")
		return du
	}

	_, id := path.Split(parsedURL.Path)
	du.ID = id

	var fileName string

	if strings.Contains(parsedURL.Host, "clips.twitch.tv") || strings.Contains(parsedURL.Path, "/clip/") {
		du.Type = TypeClip
		fileName, err = dl.getClipTitle(du.ID)
	} else if strings.Contains(parsedURL.Path, "/videos/") {
		du.Type = TypeVOD
		fileName, err = dl.getVODTitle(du.ID)

		if du.Start == 0 {
			t := parsedURL.Query().Get("t")
			if t != "" {
				du.Start, _ = time.ParseDuration(t)
			}
		}

		if du.Start > 0 && du.End > 0 && du.Start >= du.End {
			du.Error = fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL: %s", du.Start, du.End, URL)
		}
	} else {
		// add stronger checks, check lenght of path parts
		du.Type = TypeLivestream
		fileName, err = dl.getStreamTitle(du.ID)
	}

	if err != nil {
		du.Error = err
		return du
	}

	if quality == "" {
		quality = "best"
	}

	du.Quality, du.Error = QualityFromString(quality)
	if du.Error != nil {
		return du
	}

	ext := "mp4"
	if strings.HasPrefix(quality, "audio") {
		ext = "mp3"
	}

	f, err := fileutil.CreateFile(output, fileName, ext)
	if err != nil {
		du.Error = err
		return du
	}
	du.Writer = f

	return du
}

func (dl *Downloader) getVODTitle(id string) (string, error) {
	d, err := dl.TWApi.VideoMetadata(id)
	if err != nil {
		return "", err
	}
	return d.Video.Title, nil
}

func (dl *Downloader) getStreamTitle(id string) (string, error) {
	d, err := dl.TWApi.StreamMetadata(id)
	if err != nil {
		return "", err
	}
	return d.BroadcastSettings.Title, nil
}

func (dl *Downloader) getClipTitle(id string) (string, error) {
	d, err := dl.TWApi.ClipMetadata(id)
	if err != nil {
		return "", err
	}
	return d.Video.Title, nil
}
