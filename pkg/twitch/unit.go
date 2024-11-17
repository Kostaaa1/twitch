package twitch

import (
	"fmt"
	"io"
	"time"
)

type VideoType int

const (
	TypeClip VideoType = iota
	TypeVOD
	TypeLivestream
)

func (t VideoType) String() string {
	switch t {
	case TypeClip:
		return "TypeClip"
	case TypeVOD:
		return "TypeVOD"
	case TypeLivestream:
		return "TypeLivestream"
	default:
		return "Unknown"
	}
}

func GetVideoType(t string) (VideoType, error) {
	switch t {
	case "TypeClip":
		return TypeClip, nil
	case "TypeVOD":
		return TypeVOD, nil
	case "TypeLivestream":
		return TypeLivestream, nil
	default:
		return 0, fmt.Errorf("invalid video type: %s", t)
	}
}

type MediaUnit struct {
	Slug    string
	Type    VideoType
	Quality string
	Start   time.Duration
	End     time.Duration
	W       io.Writer
	Error   error
}
