package spinner

import "time"

// type unitState int

// const (
// 	unitProcessing unitState = iota
// 	unitDone
// )

type unitSize int

type unit struct {
	title string
	err   error

	downloadSize          float64
	downloadUnitSize      unitSize
	downloadSpeed         float64
	downloadSpeedUnitSize unitSize

	// TODO: handle ETA (estimated time left)
	// eta       time.Duration
	startTime time.Time
	elapsed   time.Duration

	isDone bool
}

type UnitProvider interface {
	// identifier for accessing and updating units, it needs to match the ID from the Message struct
	GetID() any
	// used as text display before progress
	GetTitle() string
	// used when initializing spinner.units. if unit has error when initializing, it will update the done count and state
	GetError() error
}

const (
	sizeB unitSize = iota
	sizeKB
	sizeMB
	sizeGB
	sizeTB
)

func (s unitSize) String() string {
	switch s {
	case sizeB:
		return "B"
	case sizeKB:
		return "KB"
	case sizeMB:
		return "MB"
	case sizeGB:
		return "GB"
	case sizeTB:
		return "TB"
	}
	return ""
}
