package twitch

import (
	"strings"
)

var (
	resolutionKeys = []string{"best", "1080p60", "720p60", "480p30", "360p30", "160p30", "worst"}
)

func GetResolution(quality string, vtype VideoType) string {
	if quality == "best" {
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

	for i, q := range resolutionKeys {
		if strings.HasPrefix(q, quality) || strings.HasPrefix(quality, q) {
			return resolutionKeys[i]
		}
	}

	return "1080p60"
}
