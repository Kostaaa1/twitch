package kick

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/google/uuid"
)

type Unit struct {
	UUID    uuid.UUID
	Channel *string
	Quality string
	Start   time.Duration
	End     time.Duration
	Error   error
	Title   string
	W       io.Writer
}

type UnitOptions func(*Unit)

func WithWriter(dir string) UnitOptions {
	return func(u *Unit) {
		u.W, u.Error = fileutil.CreateFile(dir, u.GetTitle(), "mp4")
	}
}

func WithTimestamps(start, end time.Duration) UnitOptions {
	return func(u *Unit) {
		u.Start = start
		u.End = end
	}
}

// input can be either VOD uuid or URL
func NewUnit(input, quality string, opts ...UnitOptions) *Unit {
	unit := &Unit{}

	if err := validateQuality(quality); err != nil {
		unit.Error = err
		return unit
	}

	raw, err := url.ParseRequestURI(input)
	if err == nil {
		parts := strings.Split(raw.Path, "/")

		id, err := uuid.Parse(parts[3])
		if err != nil {
			unit.Error = err
			return unit
		}

		unit.Channel = &parts[1]
		unit.UUID = id
	} else if uuid.Validate(input) == nil {
		id, _ := uuid.Parse(input)
		unit.UUID = id
	} else {
		unit.Error = fmt.Errorf("error: invalid input for kick video")
		return unit
	}

	for _, opt := range opts {
		opt(unit)
	}

	return unit
}

func (unit *Unit) NotifyProgressChannel(msg spinner.Message, ch chan spinner.Message) {
	if unit.W == nil || ch == nil {
		return
	}
	msg.ID = unit.GetID()
	ch <- msg
}

func (u *Unit) CloseWriter() error {
	if f, ok := u.W.(*os.File); ok && f != nil {
		if u.Error != nil {
			os.Remove(f.Name())
		}
		return f.Close()
	}
	return nil
}

func validateQuality(q string) error {
	valid := []string{"best", "1080", "720", "480", "360", "160", "worst"}
	for _, v := range valid {
		if strings.HasPrefix(v, q) {
			return nil
		}
	}
	return fmt.Errorf("error: invalid quality")
}

// Satisfies spinner.UnitProvider
func (u Unit) GetError() error {
	return u.Error
}

func (u Unit) GetID() any {
	return u.UUID.String()
}

func (u Unit) GetTitle() string {
	return u.UUID.String()
}
