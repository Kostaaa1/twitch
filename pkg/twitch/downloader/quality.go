package downloader

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
	Quality1080p60 QualityType = iota
	Quality720p60
	Quality480p30
	Quality360p30
	Quality160p30
	QualityAudioOnly
	QualityWorst
)

func (qt *QualityType) Downgrade() {
	if *qt == QualityWorst {
		return
	}
	*qt += 1
}

func (qt *QualityType) Upgrade() {
	if *qt == Quality1080p60 {
		return
	}
	*qt -= 1
}

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

func ParseQuality(q string) (QualityType, error) {
	switch {
	case q == "" || q == "best" || strings.HasPrefix(q, "1080"):
		return Quality1080p60, nil
	case strings.HasPrefix(q, "720"):
		return Quality720p60, nil
	case strings.HasPrefix(q, "480"):
		return Quality480p30, nil
	case strings.HasPrefix(q, "360"):
		return Quality360p30, nil
	case q == "worst" || strings.HasPrefix(q, "160"):
		return Quality160p30, nil
	case strings.HasPrefix(q, "audio"):
		return QualityAudioOnly, nil
	default:
		return 0, fmt.Errorf("invalid quality was provided: %s. valid are: %s", q, strings.Join(qualities, ", "))
	}
}
