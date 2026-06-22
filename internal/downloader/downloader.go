package downloader

import (
	"context"
	"log/slog"
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
	notifyFn func(Progress)
	log      *slog.Logger
}

func New(twClient *twitch.Client) *Downloader {
	return &Downloader{
		twClient: twClient,
		http:     http.DefaultClient,
	}
}

func (c *Downloader) SetLogger(l *slog.Logger)              { c.log = l }
func (c *Downloader) SetProgressNotifier(fn func(Progress)) { c.notifyFn = fn }

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
		ID:    u.GetID(),
		Err:   err,
		Bytes: 0,
		Done:  true,
	})

	return err
}
