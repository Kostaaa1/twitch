package httputil

import (
	"log/slog"
	"net/http"
	"time"
)

type BrowserTransport struct {
	base         http.RoundTripper
	mimicBrowser bool
	logRequest   bool
	log          *slog.Logger
}

func WithLog(log *slog.Logger) func(t *BrowserTransport) {
	return func(t *BrowserTransport) {
		t.log = log
		t.logRequest = true
	}
}

func WithBrowserMimic() func(t *BrowserTransport) {
	return func(t *BrowserTransport) { t.mimicBrowser = true }
}

func NewBrowserTransport(base http.RoundTripper, opts ...func(*BrowserTransport)) *BrowserTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	t := &BrowserTransport{base: base}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *BrowserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.mimicBrowser {
		req := req.Clone(req.Context())
		req.Header.Set("Origin", "https://www.twitch.tv")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		req.Header.Set("Referer", "https://www.twitch.tv/")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept_language", "en-US")
		// req.Header.Set("X-Device-Id", t.deviceID)
		// req.Header.Set("Client-Session-ID", "")
		// req.Header.Set("Client-Version", "")
		// req.Header.Set("Client-Integrity", "")
	}

	if !t.logRequest {
		return t.base.RoundTrip(req)
	}

	start := time.Now()
	next, err := t.base.RoundTrip(req)

	attrs := []any{
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Duration("duration", time.Since(start)),
	}

	if next != nil {
		attrs = append(attrs, slog.Int("status", next.StatusCode))
	}

	if err != nil {
		t.log.DebugContext(req.Context(), "[HTTP]: request failed", append(attrs, slog.Any("error", err))...)
		return next, err
	}

	t.log.DebugContext(req.Context(), "[HTTP]", attrs...)
	return next, err
}
