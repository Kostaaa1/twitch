package logger

import (
	"io"
	"log/slog"
)

func New(w io.Writer, vc int) *slog.Logger {
	var lvl slog.Level

	switch vc {
	case 1:
		lvl = slog.LevelInfo
	case 2:
		lvl = slog.LevelDebug
	case 3:
		lvl = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(
		w,
		&slog.HandlerOptions{
			Level: lvl,
		},
	))
}
