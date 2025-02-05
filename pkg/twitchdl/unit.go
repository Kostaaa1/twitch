package twitchdl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
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

type DownloadUnit struct {
	ID      string
	Type    VideoType
	URL     string
	Quality string
	Start   time.Duration
	End     time.Duration
	Writer  io.Writer
	Error   error
}

// DownloadUnit
// func (du *DownloadUnit) Validate() *ValidatedUnit {
// 	validated := ValidatedUnit{
// 		URL:     du.URL,
// 		Quality: du.Quality,
// 		Writer:  du.Writer,
// 		Start:   du.Start,
// 		End:     du.End,
// 		Error:   du.Error,
// 	}
// 	u, err := url.Parse(du.URL)
// 	if err != nil {
// 		validated.Error = fmt.Errorf("failed to parse the URL: %s", err)
// 	}
// 	_, id := path.Split(u.Path)
// 	validated.ID = id
// 	if !strings.Contains(u.Hostname(), "twitch.tv") {
// 		validated.Error = errors.New("the hostname of the URL does not contain twitch.tv")
// 	}
// 	if strings.Contains(u.Host, "clips.twitch.tv") || strings.Contains(u.Path, "/clip/") {
// 		validated.Type = TypeClip
// 	} else if strings.Contains(u.Path, "/videos/") {
// 		if validated.Start == 0 {
// 			t := u.Query().Get("t")
// 			if t != "" {
// 				s, err := time.ParseDuration(t)
// 				if err != nil {
// 					validated.Error = errors.New("timestamp not valid format. valid - [1h3m22s]")
// 				}
// 				validated.Start = s
// 			}
// 		}
// 		validated.Type = TypeVOD
// 	} else {
// 		validated.Type = TypeLivestream
// 	}
// 	return &validated
// }

func (mu DownloadUnit) GetError() error {
	return mu.Error
}

func (mu DownloadUnit) GetTitle() string {
	if f, ok := mu.Writer.(*os.File); ok && f != nil {
		if mu.Error != nil {
			os.Remove(f.Name())
		}
		return f.Name()
	}
	return mu.ID
}

// refactor this
func NewUnit(URL, quality, output string, start, end time.Duration) DownloadUnit {
	du := DownloadUnit{
		Start: start,
		End:   end,
	}

	u, err := url.Parse(URL)
	if err != nil {
		du.Error = err
	}

	if !strings.Contains(u.Hostname(), "twitch.tv") {
		du.Error = errors.New("the hostname of the URL does not contain twitch.tv")
	}

	if strings.Contains(u.Host, "clips.twitch.tv") || strings.Contains(u.Path, "/clip/") {
		du.Type = TypeClip
	} else if strings.Contains(u.Path, "/videos/") {
		if du.Start == 0 {
			t := u.Query().Get("t")
			if t != "" {
				s, err := time.ParseDuration(t)
				if err != nil {
					du.Error = errors.New("timestamp not valid format. valid - [1h3m22s]")
				}
				du.Start = s
			}
		}
		du.Type = TypeVOD
	} else {
		du.Type = TypeLivestream
	}

	if du.Type == TypeVOD {
		if start > 0 && end > 0 && start >= end {
			du.Error = fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL: %s", start, end, URL)
		}
	}

	quality, err = GetQuality(quality, du.Type)
	if err != nil {
		du.Error = err
	}

	fileName := fmt.Sprintf("%s_%s", du.ID, quality)
	ext := "mp4"
	if quality == "audio_only" {
		ext = "mp3"
	}

	if du.Error == nil {
		f, err := fileutil.CreateFile(output, fileName, ext)
		if err != nil {
			du.Error = err
		}
		du.Writer = f
	}

	return du
}
