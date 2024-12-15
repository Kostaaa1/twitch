package twitch

import (
	"fmt"
	"io"
	"os"
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

func (u MediaUnit) GetTitle() string {
	if file, ok := u.W.(*os.File); ok && file != nil {
		if u.Error != nil {
			os.Remove(file.Name())
		}
		return file.Name()
	}
	return u.Slug
}

func (u MediaUnit) GetError() error {
	return u.Error
}
