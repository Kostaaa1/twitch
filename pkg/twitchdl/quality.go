package twitchdl

import (
	"fmt"
	"strings"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Quality struct {
	Res string
	FPS int32
}

type QualityType int

const (
	Best QualityType = iota
	Quality1080p60
	Quality720p60
	Quality720p30
	Quality480p30
	AudioOnly
	Quality360p30
	Quality160p30
	Worst
)

var Qualities = []string{
	"best",
	"1080p60",
	"720p60",
	"720p30",
	"480p30",
	"audio_only",
	"360p30",
	"160p30",
	"worst",
}

func ValidateQuality(quality string, vtype VideoType) (string, error) {
	for _, q := range Qualities {
		if q == quality || strings.HasPrefix(quality, q) || strings.HasPrefix(q, quality) {
			if vtype == TypeVOD {
				switch quality {
				case "best", "1080p60":
					return "chunked", nil
				case "audio_only":
					return "audio_only", nil
				case "worst":
					return "160p30", nil
				}
			}
			return q, nil
		}
	}
	return "", fmt.Errorf("invalid quality was provided: %s. these are valid: %s", quality, strings.Join(Qualities, ", "))
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
