package downloader

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
