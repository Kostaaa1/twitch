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
	if f, ok := mu.Writer.(*os.File); ok && f != nil {
		if mu.Error != nil { // ??
			os.Remove(f.Name())
		}
		return f.Name()
	}
	return mu.ID
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
		du.Title, err = dl.getClipTitle(du.ID)
	} else if strings.Contains(u.Path, "/videos/") {
		du.Type = TypeVOD
		du.Title, err = dl.getVODTitle(du.ID)
		assignTimestampFromURL(&du, u)
		if du.Start > 0 && du.End > 0 && du.Start >= du.End {
			du.Error = fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL: %s", du.Start, du.End, URL)
		}
	} else {
		du.Type = TypeLivestream
		du.Title, err = dl.getStreamTitle(du.ID)
	}

	if err != nil {
		du.Error = err
		return du
	}

	du.Quality, du.Error = ValidateQuality(quality, du.Type)
	if du.Error != nil {
		return du
	}

	ext := "mp4"
	if quality == "audio_only" {
		ext = "mp3"
	}

	f, err := fileutil.CreateFile(output, du.Title, ext)
	if err != nil {
		du.Error = err
		return du
	}
	du.Writer = f

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

func (dl *Downloader) getVODTitle(id string) (string, error) {
	d, err := dl.api.VideoMetadata(id)
	if err != nil {
		return "", err
	}
	return d.Video.Title, nil
}

func (dl *Downloader) getStreamTitle(id string) (string, error) {
	d, err := dl.api.StreamMetadata(id)
	if err != nil {
		return "", err
	}
	return d.BroadcastSettings.Title, nil
}

func (dl *Downloader) getClipTitle(id string) (string, error) {
	d, err := dl.api.ClipMetadata(id)
	if err != nil {
		return "", err
	}
	return d.Video.Title, nil
}
