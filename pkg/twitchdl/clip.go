package twitchdl

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Kostaaa1/twitch/internal/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func (dl *Downloader) GetClipVideoURL(clip twitch.Clip, quality string) (string, error) {
	sourceURL := extractClipSourceURL(clip.Assets[0].VideoQualities, quality)
	usherURL, err := dl.api.ConstructUsherURL(clip.PlaybackAccessToken, sourceURL)
	if err != nil {
		return "", err
	}
	return usherURL, nil
}

func (mu DownloadUnit) downloadClip(dl *Downloader) error {
	clip, err := dl.api.ClipData(mu.ID)
	if err != nil {
		return err
	}

	usherURL, err := dl.GetClipVideoURL(clip, mu.Quality)
	if err != nil {
		return err
	}

	var writtenBytes int64
	if mu.Quality == "audio_only" {
		writtenBytes, err = extractAudio(usherURL, mu.Writer)
	} else {
		writtenBytes, err = dl.downloadFromURL(usherURL, mu.Writer)
	}

	if err != nil {
		return err
	}

	if file, ok := mu.Writer.(*os.File); ok && file != nil {
		dl.progressCh <- spinner.ChannelMessage{
			Text:  file.Name(),
			Bytes: writtenBytes,
		}
	}

	return nil
}

func extractAudio(segmentURL string, w io.Writer) (int64, error) {
	cmd := exec.Command("ffmpeg", "-i", segmentURL, "-q:a", "0", "-map", "a", "-f", "mp3", "-")
	cmd.Stdout = nil
	cmd.Stderr = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	n, err := io.Copy(w, stdout)
	if err != nil {
		return 0, fmt.Errorf("failed to copy audio data: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return 0, fmt.Errorf("FFmpeg conversion failed: %w", err)
	}
	return n, nil
}
