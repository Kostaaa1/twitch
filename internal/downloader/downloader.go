package downloader

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
)

type Progress struct {
	ID    string
	Label string
	Bytes int64
	Error error
	Done  bool
	Total float64
}

type Downloader struct {
	gql      *gql.Client
	http     *http.Client
	notifyFn func(Progress)
}

func New(gql *gql.Client, http *http.Client) *Downloader {
	return &Downloader{gql: gql, http: http}
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
		ID:    u.GetID(),
		Label: u.GetLabel(),
		Error: err,
		Bytes: 0,
		Total: 0,
		Done:  true,
	})

	return err
}

func (dl *Downloader) download(u *Unit, r io.ReadCloser) error {
	defer r.Close()

	if u.w == nil {
		if err := dl.openFile(context.Background(), u); err != nil {
			return err
		}
		// return errors.New("missing writer")
	}

	n, err := io.Copy(u.w, r)
	if err != nil {
		return err
	}

	dl.notify(Progress{
		ID:    u.GetID(),
		Label: u.GetLabel(),
		Bytes: n,
		Done:  false,
		Error: nil,
		Total: 0,
	})

	return nil
}

func (dl *Downloader) segmentFetchDownload(ctx context.Context, u *Unit, segURL string) error {
	body, err := dl.fetchSegment(ctx, segURL)
	if err != nil {
		return err
	}
	return dl.download(u, body)
}

func (dl *Downloader) fetchTitle(ctx context.Context, u *Unit) (title string, err error) {
	switch u.Type {
	case TypeClip:
		title, err = dl.gql.ClipTitle(ctx, u.ID)
	case TypeVOD:
		title, err = dl.gql.VideoTitle(ctx, u.ID)
	case TypeLivestream:
		title, err = dl.gql.StreamTitle(ctx, u.ID)
	}
	return
}

func (dl *Downloader) openFile(ctx context.Context, u *Unit) error {
	if u.dir == "" {
		return errors.New("missing dir")
	}
	if u.ext == "" {
		return errors.New("missing file extension")
	}

	if u.filename == "" {
		title, err := dl.fetchTitle(ctx, u)
		if err != nil {
			return err
		}
		u.Title = title
		u.filename = u.Title
	}

	pathname, err := fileutil.ConstructPathname(u.dir, u.filename, u.ext)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(pathname, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	u.w = f

	return nil
}
