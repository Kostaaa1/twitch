package twitchdl

import (
	"os"
	"strings"

	"github.com/Kostaaa1/twitch/internal/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func (mu DownloadUnit) downloadClip(dl *Downloader) error {
	clip, err := dl.TWApi.ClipMetadata(mu.ID)
	if err != nil {
		return err
	}

	usherURL, err := dl.GetClipVideoURL(clip, mu.Quality)
	if err != nil {
		return err
	}

	var writtenBytes int64
	if mu.Quality == "audio_only" {
		// writtenBytes, err = extractAudio(usherURL, mu.Writer)
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

// func extractAudio(segmentURL string, w io.Writer) (int64, error) {
// 	cmd := exec.Command("ffmpeg", "-i", segmentURL, "-q:a", "0", "-map", "a", "-f", "mp3", "-")
// 	cmd.Stdout = nil
// 	cmd.Stderr = nil
// 	stdout, err := cmd.StdoutPipe()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to get stdout pipe: %w", err)
// 	}
// 	if err := cmd.Start(); err != nil {
// 		return 0, fmt.Errorf("failed to start FFmpeg: %w", err)
// 	}
// 	n, err := io.Copy(w, stdout)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to copy audio data: %w", err)
// 	}
// 	if err := cmd.Wait(); err != nil {
// 		return 0, fmt.Errorf("FFmpeg conversion failed: %w", err)
// 	}
// 	return n, nil
// }

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
	for i, val := range Qualities {
		if val == quality {
			id = i
		}
	}
	if id > 0 {
		return extractClipSourceURL(videoQualities, Qualities[id-1])
	} else {
		return extractClipSourceURL(videoQualities, Qualities[id+1])
	}
}

func (dl *Downloader) GetClipVideoURL(clip twitch.Clip, quality string) (string, error) {
	sourceURL := extractClipSourceURL(clip.Assets[0].VideoQualities, quality)
	usherURL, err := dl.TWApi.ConstructUsherURL(clip.PlaybackAccessToken, sourceURL)
	if err != nil {
		return "", err
	}
	return usherURL, nil
}
