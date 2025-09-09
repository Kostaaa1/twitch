package spinner

import "time"

type unit struct {
	title      string
	err        error
	totalBytes float64
	startTime  time.Time
	elapsed    time.Duration
	isDone     bool
}

type UnitProvider interface {
	// identifier for accessing and updating units, it needs to match the ID from the Message struct
	GetID() any
	// used as text display before progress
	GetTitle() string
	// used when initializing spinner.units. if unit has error when initializing, it will update the done count and state
	GetError() error
}
