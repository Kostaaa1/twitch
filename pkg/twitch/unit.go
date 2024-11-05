package twitch

import (
	"io"
	"time"
)

type VideoType int

const (
	TypeClip VideoType = iota
	TypeVOD
	TypeLivestream
)

type MediaUnit struct {
	Slug    string
	Type    VideoType
	Quality string
	Start   time.Duration
	End     time.Duration
	W       io.Writer
}

func (unit *MediaUnit) SetWriter(w io.Writer) {
	unit.W = w
}
