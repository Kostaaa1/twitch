package twitchdl

import (
	"fmt"
	"strings"
)

type Quality struct {
	Res string
	FPS int32
}

type QualityType int

const (
	QualityBest QualityType = iota
	Quality1080p60
	Quality720p60
	Quality480p30
	Quality360p30
	Quality160p30
	QualityAudioOnly
	QualityWorst
)

func (qt QualityType) String() string {
	switch qt {
	case Quality1080p60:
		return "1080p60"
	case Quality720p60:
		return "720p60"
	case Quality480p30:
		return "480p30"
	case Quality360p30:
		return "360p30"
	case Quality160p30:
		return "160p30"
	case QualityAudioOnly:
		return "audio_only"
	default:
		return ""
	}
}

func QualityFromInput(quality string) (QualityType, error) {
	switch {
	case quality == "best" || strings.HasPrefix(quality, "1080"):
		return Quality1080p60, nil
	case strings.HasPrefix(quality, "720"):
		return Quality720p60, nil
	case strings.HasPrefix(quality, "480"):
		return Quality480p30, nil
	case strings.HasPrefix(quality, "360"):
		return Quality360p30, nil
	case quality == "worst" || strings.HasPrefix(quality, "160"):
		return Quality160p30, nil
	case strings.HasPrefix(quality, "audio"):
		return QualityAudioOnly, nil
	default:
		return 0, fmt.Errorf("invalid quality was provided: %s. valid are: %s", quality, strings.Join(qualities, ", "))
	}
}

var qualities = []string{
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

// func ValidateQuality(quality string, vtype VideoType) (string, error) {
// 	for _, q := range qualities {
// 		if q == quality || strings.HasPrefix(quality, q) || strings.HasPrefix(q, quality) {
// 			if vtype == TypeVOD {
// 				switch quality {
// 				case "best", "1080p60":
// 					return "chunked", nil
// 				case "audio_only":
// 					return "audio_only", nil
// 				case "worst":
// 					return "160p30", nil
// 				}
// 			}
// 			return q, nil
// 		}
// 	}
// 	return "", fmt.Errorf("invalid quality was provided: %s. these are valid: %s", quality, strings.Join(qualities, ", "))
// }
