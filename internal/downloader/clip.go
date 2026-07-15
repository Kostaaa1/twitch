package downloader

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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

func (dl *Downloader) downloadClip(ctx context.Context, u *Unit) error {
	clip, err := dl.gql.ClipMetadata(ctx, u.ID)
	if err != nil {
		return err
	}
	at := clip.PlaybackAccessToken

	clipSrc := extractClipSourceURL(clip.Assets[0].VideoQualities, u.Quality.String())
	signedURL := fmt.Sprintf("%s?sig=%s&token=%s", clipSrc, url.QueryEscape(at.Signature), url.QueryEscape(at.Value))

	if err := u.setFileExt(signedURL); err != nil {
		return err
	}

	if u.Quality == QualityAudioOnly {
		// n, err = extractAudio(usherURL, unit.Writer)
		return errors.New("audio quality for clip not supported yet")
	} else {
		if err := dl.fetchDownload(ctx, u, signedURL); err != nil {
			return err
		}
	}

	return nil
}
