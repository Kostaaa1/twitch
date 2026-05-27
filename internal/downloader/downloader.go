package downloader

import (
	"context"
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Progress struct {
	ID    string
	Bytes int64
	Err   error
	Done  bool
}

type Downloader struct {
	twClient *twitch.Client
	http     *http.Client
	config   *Config
	notifyFn func(Progress)
}

type Config struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
}

func New(twClient *twitch.Client, conf *Config) *Downloader {
	return &Downloader{
		twClient: twClient,
		config:   conf,
		http:     http.DefaultClient,
	}
}

func (c *Downloader) SetProgressNotifier(fn func(Progress)) {
	c.notifyFn = fn
}

func (c *Downloader) notify(msg Progress) {
	if c.notifyFn != nil {
		c.notifyFn(msg)
	}
}

func (dl *Downloader) Download(ctx context.Context, u Unit) error {
	defer u.CloseWriter()

	err := u.Error

	if err != nil {
		dl.notify(Progress{
			ID:    u.GetID(),
			Err:   err,
			Bytes: 0,
			Done:  true,
		})
		return err
	}

	switch u.Type {
	case TypeVOD:
		err = dl.downloadVOD(ctx, u)
	case TypeClip:
		err = dl.downloadClip(ctx, u)
	case TypeLivestream:
		err = dl.recordLivestream(ctx, u)
	}

	dl.notify(Progress{
		ID:    u.GetID(),
		Err:   err,
		Bytes: 0,
		Done:  true,
	})

	return err
}
