package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type ProgressMessage struct {
	ID    any
	Bytes int64
	Err   error
	Done  bool
}

type Downloader struct {
	twClient *twitch.Client
	config   Config
	// TODO: this should not depend on spinner package
	// progCh chan spinner.Message
	notifyFn func(ProgressMessage)
}

type Config struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
}

func New(twClient *twitch.Client, conf Config) *Downloader {
	return &Downloader{
		twClient: twClient,
		config:   conf,
	}
}

func (c *Downloader) SetProgressNotifier(fn func(ProgressMessage)) {
	c.notifyFn = fn
}

func (c *Downloader) notify(msg ProgressMessage) {
	if c.notifyFn != nil {
		c.notifyFn(msg)
	}
}

func (dl *Downloader) Download(ctx context.Context, u Unit) error {
	defer u.CloseWriter()

	if u.Error != nil {
		return u.Error
	}

	switch u.Type {
	case TypeVOD:
		u.Error = dl.downloadVOD(ctx, u)
	case TypeClip:
		u.Error = dl.downloadClip(ctx, u)
	case TypeLivestream:
		u.Error = dl.recordStream(ctx, u)
	}

	if u.Error != nil {
		dl.notify(ProgressMessage{
			ID:    u.GetID(),
			Err:   u.Error,
			Bytes: 0,
			Done:  true,
		})
		return u.Error
	}

	return nil
}

func (dl *Downloader) fetch(ctx context.Context, url string) (io.ReadCloser, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := dl.twClient.HttpClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, resp.StatusCode, err
}
