package log

import (
	"log/slog"
	"os"
)

func New(lvl int) *slog.Logger {
	w := os.Stdout
	var level slog.Level

	switch lvl {
	case 0:
		w = os.Stdout
	case 1:
		level = slog.LevelInfo
	case 2:
		level = slog.LevelError
	case 3:
		level = slog.LevelDebug
	}

	log := slog.New(slog.NewTextHandler(
		w,
		&slog.HandlerOptions{
			Level: level,
			// ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 	if ok := a.(slog.Time); ok {
			// 	}
			// 	return a
			// },
		},
	))

	return log
}
