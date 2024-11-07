package twitch

import "strings"

var (
	formatKeys = []string{"best", "1080p60", "720p60", "480p30", "360p30", "160p30", "worst", "audio_only"}
)

func extractClipSourceURL(videoQualities []VideoQuality, quality string) string {
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
		return extractClipSourceURL(videoQualities, formatKeys[id-1])
	}
	return "best"
}

func GetFormat(quality string, vtype VideoType) string {
	if vtype == TypeVOD && strings.HasPrefix(quality, "audio") {
		return "audio_only"
	}
	if quality == "" || quality == "best" || strings.HasPrefix(quality, "1080") {
		if vtype == TypeVOD {
			return "chunked"
		}
		return "best"
	}

	if quality == "worst" {
		if vtype == TypeVOD {
			return "16030"
		}
		return "worst"
	}

	for i, q := range formatKeys {
		if strings.HasPrefix(q, quality) || strings.HasPrefix(quality, q) {
			return formatKeys[i]
		}
	}

	id := getFormatId(quality)
	if id > 0 {
		return GetFormat(formatKeys[id-1], vtype)
	} else {
		return GetFormat(formatKeys[id+1], vtype)
	}
}

func getFormatId(quality string) int {
	for i, val := range formatKeys {
		if val == quality {
			return i
		}
	}
	return -1
}
