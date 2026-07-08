package downloader

import (
	"context"
	"errors"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
)

func extractClipSourceURL(videoQualities []gql.VideoQuality, quality string) string {
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

func (dl *Downloader) downloadClip(ctx context.Context, unit *Unit) error {
	clip, err := dl.gql.ClipMetadata(ctx, unit.ID)
	if err != nil {
		return err
	}

	clipDataURL := extractClipSourceURL(clip.Assets[0].VideoQualities, unit.Quality.String())

	usherURL, err := dl.gql.ConstructUsherURL(clip.PlaybackAccessToken, clipDataURL)
	if err != nil {
		return err
	}

	if unit.Quality == QualityAudioOnly {
		// n, err = extractAudio(usherURL, unit.Writer)
		return errors.New("audio quality for clip not yet supported")
	} else {
		if err := dl.segmentFetchDownload(ctx, unit, usherURL); err != nil {
			return err
		}
	}

	return nil
}
