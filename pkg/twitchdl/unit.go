package twitchdl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
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

type DownloadUnit struct {
	ID      string
	Title   string
	Type    VideoType
	URL     string
	Quality string
	Start   time.Duration
	End     time.Duration
	Writer  io.Writer
	Error   error
}

func (mu DownloadUnit) GetError() error {
	return mu.Error
}

func (mu DownloadUnit) GetTitle() string {
	return mu.Title
	// if f, ok := mu.Writer.(*os.File); ok && f != nil {
	// 	if mu.Error != nil { // ??
	// 		os.Remove(f.Name())
	// 	}
	// 	return f.Name()
	// }
	// return mu.ID
}

func (dl *Downloader) NewUnit(URL, quality, output string, start, end time.Duration) DownloadUnit {
	du := DownloadUnit{
		Start: start,
		End:   end,
	}

	u, err := url.Parse(URL)
	if err != nil {
		du.Error = err
		return du
	}

	if !strings.Contains(u.Hostname(), "twitch.tv") {
		du.Error = errors.New("the hostname of the URL does not contain twitch.tv")
		return du
	}

	_, du.ID = path.Split(u.Path)

	if strings.Contains(u.Host, "clips.twitch.tv") || strings.Contains(u.Path, "/clip/") {
		du.Type = TypeClip
		clip, err := dl.api.Clip(du.ID)

		if err != nil {
			du.Error = err
			return du
		}

		du.Title = clip.Video.Title
	} else if strings.Contains(u.Path, "/videos/") {
		assignTimestampFromURL(&du, u)
		if du.Start > 0 && du.End > 0 && du.Start >= du.End {
			du.Error = fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL: %s", du.Start, du.End, URL)
			return du
		}

		du.Type = TypeVOD
		vod, err := dl.api.VideoMetadata(du.ID)
		if err != nil {
			du.Error = err
			return du
		}
		du.Title = vod.Video.Title
	} else {
		stream, err := dl.api.StreamMetadata(du.ID)
		if err != nil {
			du.Error = err
			return du
		}
		du.Title = stream.BroadcastSettings.Title
		du.Type = TypeLivestream
	}

	du.Quality, du.Error = ValidateQuality(quality, du.Type)

	ext := "mp4"
	if quality == "audio_only" {
		ext = "mp3"
	}

	if du.Error == nil {
		f, err := fileutil.CreateFile(output, du.Title, ext)
		if err != nil {
			du.Error = err
			return du
		}
		du.Writer = f
	}

	return du
}

func assignTimestampFromURL(du *DownloadUnit, u *url.URL) {
	if du.Start == 0 {
		t := u.Query().Get("t")
		if t != "" {
			du.Start, _ = time.ParseDuration(t)
		}
	}
}
