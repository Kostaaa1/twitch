package downloader

import (
	"context"
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Progress struct {
	Label string
	Bytes int64
	Error error
	Done  bool
	Total float64
}

type Downloader struct {
	twClient *twitch.Client
	http     *http.Client
	notifyFn func(Progress)
}

func New(twClient *twitch.Client, http *http.Client) *Downloader {
	return &Downloader{twClient: twClient, http: http}
}

func (c *Downloader) SetProgressNotifier(fn func(Progress)) {
	c.notifyFn = fn
}

func (c *Downloader) notify(msg Progress) {
	if c.notifyFn != nil {
		c.notifyFn(msg)
	}
}

func (dl *Downloader) Download(ctx context.Context, u *Unit) error {
	defer u.CloseWriter()

	var err error

	switch u.Type {
	case TypeVOD:
		err = dl.downloadVOD(ctx, u)
	case TypeClip:
		err = dl.downloadClip(ctx, u)
	case TypeLivestream:
		err = dl.recordLivestream(ctx, u)
	}

	dl.notify(Progress{
		Label: u.GetLabel(),
		Error: err,
		Bytes: 0,
		Done:  true,
	})

	return err
}
