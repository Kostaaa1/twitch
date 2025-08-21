package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Downloader struct {
	twClient   *twitch.Client
	progressCh chan spinner.ChannelMessage
	config     Config
	ctx        context.Context
}

type Config struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
	SpinnerModel    string `json:"spinner_model"`
}

func New(ctx context.Context, twClient *twitch.Client, conf Config) *Downloader {
	return &Downloader{
		ctx:      ctx,
		twClient: twClient,
		config:   conf,
	}
}

func (dl *Downloader) SetProgressChannel(progressCh chan spinner.ChannelMessage) {
	dl.progressCh = progressCh
}

func (dl *Downloader) Download(u Unit) error {
	defer u.CloseWriter()

	if u.Error == nil {
		switch u.Type {
		case TypeVOD:
			u.Error = dl.downloadVOD(u)
		case TypeClip:
			u.Error = dl.downloadClip(u)
		case TypeLivestream:
			u.Error = dl.recordStream(u)
		}
	}

	return u.Error
}

func (dl *Downloader) fetch(url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(dl.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request with context: %v", err)
	}

	resp, err := dl.twClient.HTTPClient().Do(req)
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

	resp, err := dl.twClient.HTTPClient().Do(req)
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
