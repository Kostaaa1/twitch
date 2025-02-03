package twitchdl

import (
	"os"

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

func (mu MediaUnit) downloadClip(dl *Downloader) error {
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
		writtenBytes, err = dl.downloadAndWriteSegment(usherURL, mu.Writer)
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
