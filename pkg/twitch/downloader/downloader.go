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
	// ctx      context.Context
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

// func (dl *Downloader) SetProgressChannel(progCh chan spinner.Message) {
// 	dl.progCh = progCh
// }

// func (dl *Downloader) NotifyProgressChannel(msg spinner.Message, unit Unit) {
// 	if dl.progCh == nil {
// 		return
// 	}
// 	_, ok := unit.Writer.(*os.File)
// 	if !ok {
// 		return
// 	}
// 	select {
// 	case <-dl.ctx.Done():
// 		return
// 	default:
// 		dl.progCh <- msg
// 	}
// }

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

	return nil
}

func (dl *Downloader) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := dl.twClient.HttpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (dl *Downloader) fetchWithStatus(ctx context.Context, url string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := dl.twClient.HttpClient().Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return http.StatusOK, b, err
}

func (dl *Downloader) fetchWithStatusCloser(ctx context.Context, url string) (int, io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := dl.twClient.HttpClient().Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	return http.StatusOK, resp.Body, err
}
