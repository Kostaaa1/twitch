package spinner

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

var (
	sizeUnits = []string{"B", "KB", "MB", "GB", "TB"}
	spinners  = map[string]spinner.Spinner{
		"meter":     spinner.Meter,
		"line":      spinner.Line,
		"pulse":     spinner.Pulse,
		"ellipsis":  spinner.Ellipsis,
		"jump":      spinner.Jump,
		"points":    spinner.Points,
		"globe":     spinner.Globe,
		"hamburger": spinner.Hamburger,
		"minidot":   spinner.MiniDot,
		"monkey":    spinner.Monkey,
		"moon":      spinner.Moon,
		"dot": {
			Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
			FPS:    time.Second / 10,
		},
	}
)

func validateSpinnerModel(model string) spinner.Spinner {
	_, ok := spinners[model]
	if ok {
		return spinners[model]
	} else {
		return spinners["dot"]
	}
}
