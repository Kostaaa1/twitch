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

var (
	Qualities = []string{
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
)

func getVODQuality(quality string) string {
	switch {
	case quality == "best":
		return "chunked"
	case quality == "1080p60":
		return "chunked"
	case strings.HasPrefix(quality, "audio"):
		return "audio_only"
	case quality == "worst":
		return "160p30"
	default:
		return quality
	}
}

func GetQuality(quality string, vtype VideoType) (string, error) {
	for _, q := range Qualities {
		if q == quality || strings.HasPrefix(quality, q) || strings.HasPrefix(q, quality) {
			if vtype == TypeVOD {
				return getVODQuality(q), nil
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

	id := getFormatId(quality)

	if id > 0 {
		return extractClipSourceURL(videoQualities, Qualities[id-1])
	} else {
		return extractClipSourceURL(videoQualities, Qualities[id+1])
	}
}

func getFormatId(quality string) int {
	for i, val := range Qualities {
		if val == quality {
			return i
		}
	}
	return -1
}

func IsQualityValid(quality string) bool {
	for _, q := range Qualities {
		if q == quality || strings.HasPrefix(quality, q) || strings.HasPrefix(q, quality) {
			return true
		}
	}
	return false
}
