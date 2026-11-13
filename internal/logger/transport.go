package logger

import (
	"log/slog"
	"net/http"
	"time"
)

type LoggingTransport struct {
	next http.RoundTripper
	log  *slog.Logger
}

func NewLoggingTransport(next http.RoundTripper, log *slog.Logger) *LoggingTransport {
	if next == nil {
		next = http.DefaultTransport
	}
	if log == nil {
		log = slog.Default()
	}
	return &LoggingTransport{}
}

func (l *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	next, err := l.next.RoundTrip(req)

	attrs := []any{
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Int("status", next.StatusCode),
		slog.Duration("duration", time.Since(start)),
	}

	if err != nil {
		l.log.DebugContext(req.Context(), "[HTTP]: request failed", append(attrs, slog.Any("error", err))...)
		return nil, err
	}

	l.log.DebugContext(req.Context(), "[HTTP]", attrs...)
	return nil, err
}
