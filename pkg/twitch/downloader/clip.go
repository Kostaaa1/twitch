package downloader

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func (dl *Downloader) downloadClip(unit Unit) error {
	clip, err := dl.twClient.ClipMetadata(unit.ID)
	if err != nil {
		return err
	}

	usherURL, err := dl.clipVideoURL(clip.PlaybackAccessToken, clip.Assets[0].VideoQualities, unit.Quality.String())

	if err != nil {
		return err
	}

	var n int64
	if unit.Quality == QualityAudioOnly {
		n, err = extractAudio(usherURL, unit.Writer)
	} else {
		res, err := dl.twClient.HttpClient().Get(usherURL)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		n, err = io.Copy(unit.Writer, res.Body)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	msg := spinner.ChannelMessage{Bytes: n}
	unit.NotifyProgressChannel(msg, dl.progressCh)

	return nil
}

func extractClipSourceURL(videoQualities []twitch.VideoQuality, quality string) string {
	if quality == "best" {
		return videoQualities[0].SourceURL
	}
	if quality == "worst" {
		return videoQualities[len(videoQualities)-1].SourceURL
	}

	for _, q := range videoQualities {
		if strings.HasPrefix(quality, q.Quality) || strings.HasPrefix(q.Quality, quality) {
			return q.SourceURL
		}
	}

	id := -1
	for i, val := range qualities {
		if val == quality {
			id = i
		}
	}

	if id > 0 {
		return extractClipSourceURL(videoQualities, qualities[id-1])
	} else {
		return extractClipSourceURL(videoQualities, qualities[id+1])
	}
}

func (dl *Downloader) clipVideoURL(token twitch.PlaybackAccessToken, qualities []twitch.VideoQuality, quality string) (string, error) {
	sourceURL := extractClipSourceURL(qualities, quality)
	usherURL, err := dl.twClient.ConstructUsherURL(token, sourceURL)
	if err != nil {
		return "", err
	}
	return usherURL, nil
}

// uses ffmpeg for getting the audio from a segment
func extractAudio(url string, w io.Writer) (int64, error) {
	cmd := exec.Command("ffmpeg", "-i", url, "-q:a", "0", "-map", "a", "-f", "mp3", "-")
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
