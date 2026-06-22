package spinner

import "time"

type unit struct {
	title      string
	err        error
	totalBytes float64
	startTime  time.Time
	elapsed    time.Duration
	done       bool
}

type UnitProvider interface {
	GetID() string
	// GetError() error
}
