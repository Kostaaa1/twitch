package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Downloader struct {
	twClient *twitch.Client
	config   Config
	// TODO: this should not depend on spinner package
	progCh chan spinner.Message
	ctx    context.Context
}

type Config struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
}

func New(ctx context.Context, twClient *twitch.Client, conf Config) *Downloader {
	return &Downloader{
		ctx:      ctx,
		twClient: twClient,
		config:   conf,
	}
}

func (dl *Downloader) SetProgressChannel(progCh chan spinner.Message) {
	dl.progCh = progCh
}

func (dl *Downloader) NotifyProgressChannel(msg spinner.Message, unit Unit) {
	if dl.progCh == nil {
		return
	}

	_, ok := unit.Writer.(*os.File)
	if !ok {
		return
	}

	select {
	case <-dl.ctx.Done():
		return
	default:
		dl.progCh <- msg
	}
}

func (dl *Downloader) Download(ctx context.Context, u Unit) error {
	defer u.CloseWriter()

	if u.Error != nil {
		return u.Error
	}

	switch u.Type {
	case TypeVOD:
		u.Error = dl.downloadVOD(u)
	case TypeClip:
		u.Error = dl.downloadClip(u)
	case TypeLivestream:
		u.Error = dl.recordStream(u)
	}

	return nil
}

func (dl *Downloader) fetch(url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(dl.ctx, http.MethodGet, url, nil)
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

func (dl *Downloader) fetchWithStatus(url string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(dl.ctx, http.MethodGet, url, nil)
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
