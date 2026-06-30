package eventsub

import (
	"errors"
)

type Condition map[string]string

type Event struct {
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Condition Condition `json:"condition"`
	Transport transport `json:"transport"`
}

func (e *Event) Validate() error {
	if e.Type == "" {
		return errors.New("missing type on event")
	}
	if e.Version == "" {
		return errors.New("missing version on event")
	}
	if e.Condition == nil {
		return errors.New("missing condition JSON object on event")
	}
	if e.Transport.Method == "" {
		return errors.New("missing transport method on event")
	}
	return nil
}
