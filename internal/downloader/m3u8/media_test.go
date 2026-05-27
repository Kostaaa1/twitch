package m3u8

import (
	_ "embed"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/media_vod.m3u8
var mediaVOD string

//go:embed testdata/media_live_with_ads.m3u8
var mediaLiveWithAds string

func parseMedia(t *testing.T, s string) (*MediaPlaylist, error) {
	t.Helper()
	return ParseMediaPlaylist(strings.NewReader(s))
}

func TestParseMedia_VOD_HeaderFields(t *testing.T) {
	m, err := parseMedia(t, mediaVOD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Version != 3 {
		t.Errorf("Version: got %d, want 3", m.Version)
	}
	if m.TargetDuration != 10 {
		t.Errorf("TargetDuration: got %v, want 10", m.TargetDuration)
	}
	if m.PlaylistType != "VOD" {
		t.Errorf("PlaylistType: got %q, want %q", m.PlaylistType, "VOD")
	}
	if m.ElapsedSecs != 0.0 {
		t.Errorf("ElapsedSecs: got %v, want 0", m.ElapsedSecs)
	}
	if m.TotalSecs != 60.0 {
		t.Errorf("TotalSecs: got %v, want 60", m.TotalSecs)
	}
	if m.Timestamp != "2024-05-01T12:00:00" {
		t.Errorf("Timestamp: got %q", m.Timestamp)
	}
}

func TestParseMedia_VOD_Segments(t *testing.T) {
	m, err := parseMedia(t, mediaVOD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Segments) != 6 {
		t.Fatalf("expected 6 segments, got %d", len(m.Segments))
	}

	for i := 0; i < 6; i++ {
		wantURL := []string{"0.ts", "1.ts", "2.ts", "3.ts", "4.ts", "5.ts"}[i]
		if m.Segments[i].URL != wantURL {
			t.Errorf("segment[%d].URL: got %q, want %q", i, m.Segments[i].URL, wantURL)
		}
		if m.Segments[i].Duration != 10*time.Second {
			t.Errorf("segment[%d].Duration: got %v, want 10s", i, m.Segments[i].Duration)
		}
	}
}

// Catches the bug at media.go:108: `trimmed := v[:len(v)-1]` assumes EXTINF
// always ends with a trailing comma and empty title. Twitch live playlists
// emit `#EXTINF:2.000,live` — the title is "live", and parsing breaks today.
func TestParseMedia_Live_EXTINFWithLiveTitle(t *testing.T) {
	m, err := parseMedia(t, mediaLiveWithAds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Segments) == 0 {
		t.Fatal("no segments parsed from live playlist")
	}
	for i, seg := range m.Segments {
		if seg.Duration != 2*time.Second {
			t.Errorf("segment[%d].Duration: got %v, want 2s", i, seg.Duration)
		}
	}
}

func TestParseMedia_Live_HeaderFields(t *testing.T) {
	m, err := parseMedia(t, mediaLiveWithAds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Version != 6 {
		t.Errorf("Version: got %d, want 6", m.Version)
	}
	if m.TargetDuration != 6 {
		t.Errorf("TargetDuration: got %v, want 6", m.TargetDuration)
	}
	if m.ElapsedSecs != 600.0 {
		t.Errorf("ElapsedSecs: got %v, want 600", m.ElapsedSecs)
	}
	if m.TotalSecs != 614.0 {
		t.Errorf("TotalSecs: got %v, want 614", m.TotalSecs)
	}
}

// Live playlist has 7 segments split across two discontinuities (ad break +
// content resume). Parser should keep all of them.
func TestParseMedia_Live_KeepsSegmentsAcrossDiscontinuities(t *testing.T) {
	m, err := parseMedia(t, mediaLiveWithAds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Segments) != 7 {
		t.Fatalf("expected 7 segments across discontinuities, got %d", len(m.Segments))
	}
	wantURLs := []string{"100.ts", "101.ts", "102.ts", "103.ts", "104.ts", "105.ts", "106.ts"}
	for i, want := range wantURLs {
		if m.Segments[i].URL != want {
			t.Errorf("segment[%d].URL: got %q, want %q", i, m.Segments[i].URL, want)
		}
	}
}

// Feature: the parser ought to expose which segments were inside an ad break
// so callers can choose to skip them. Skipped today.
func TestParseMedia_Live_FlagsAdSegments(t *testing.T) {
	t.Skip("feature: mark segments between #EXT-X-DISCONTINUITY + twitch-stitched-ad DATERANGE as ads")
}

func TestParseMedia_HandlesCRLFLineEndings(t *testing.T) {
	playlist := strings.ReplaceAll(mediaVOD, "\n", "\r\n")
	m, err := parseMedia(t, playlist)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Segments) != 6 {
		t.Errorf("CRLF playlist: got %d segments, want 6", len(m.Segments))
	}
	if m.Segments[0].URL != "0.ts" {
		t.Errorf("CRLF playlist: segment URL not trimmed: %q", m.Segments[0].URL)
	}
}

func TestParseMedia_EmptyInputReturnsEmptyPlaylist(t *testing.T) {
	m, err := parseMedia(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("got nil playlist for empty input")
	}
	if len(m.Segments) != 0 {
		t.Errorf("expected 0 segments, got %d", len(m.Segments))
	}
}

func TestParseMedia_RejectsNonM3UInput(t *testing.T) {
	t.Skip("API improvement: parser should return an error for input lacking #EXTM3U")

	_, err := parseMedia(t, "garbage\nmore garbage\n")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestParseMedia_MalformedEXTINFIsReportedNotPanicked(t *testing.T) {
	playlist := `#EXTM3U
#EXTINF:not-a-number,
0.ts
`
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("parser panicked: %v", r)
		}
	}()
	_, err := parseMedia(t, playlist)
	if err == nil {
		t.Error("expected error for malformed EXTINF duration, got nil")
	}
}

// Truncate operates on a built MediaPlaylist; using an in-memory one keeps the
// math obvious. The fixture-driven tests above already cover real parser input.

func mediaWithSegments(durations ...time.Duration) *MediaPlaylist {
	m := &MediaPlaylist{}
	for i, d := range durations {
		m.Segments = append(m.Segments, Segment{
			URL:      string(rune('a'+i)) + ".ts",
			Duration: d,
		})
	}
	return m
}

func TestTruncate_PicksWindow(t *testing.T) {
	m := mediaWithSegments(10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second)
	m.Truncate(15*time.Second, 35*time.Second)

	if len(m.Segments) == 0 {
		t.Fatal("Truncate dropped every segment")
	}

	first := m.Segments[0].URL
	last := m.Segments[len(m.Segments)-1].URL
	// Window 15s..35s should overlap segments b (10-20), c (20-30), d (30-40).
	if first != "b.ts" {
		t.Errorf("first segment in window: got %q, want %q", first, "b.ts")
	}
	if last != "d.ts" {
		t.Errorf("last segment in window: got %q, want %q", last, "d.ts")
	}
}

func TestTruncate_StartZeroKeepsFirstSegment(t *testing.T) {
	m := mediaWithSegments(10*time.Second, 10*time.Second, 10*time.Second)
	m.Truncate(0, 15*time.Second)

	if len(m.Segments) == 0 {
		t.Fatal("Truncate dropped every segment")
	}
	if m.Segments[0].URL != "a.ts" {
		t.Errorf("first segment: got %q, want %q", m.Segments[0].URL, "a.ts")
	}
}

func TestTruncate_InvalidRangeIsNoop(t *testing.T) {
	m := mediaWithSegments(10*time.Second, 10*time.Second, 10*time.Second)
	before := len(m.Segments)
	m.Truncate(20*time.Second, 5*time.Second) // start > end
	if len(m.Segments) != before {
		t.Errorf("invalid range mutated playlist: got %d segments, want %d", len(m.Segments), before)
	}
}

func TestTruncate_EndBeyondPlaylistKeepsTail(t *testing.T) {
	m := mediaWithSegments(10*time.Second, 10*time.Second, 10*time.Second)
	m.Truncate(15*time.Second, 1*time.Hour)

	if len(m.Segments) == 0 {
		t.Fatal("Truncate dropped every segment")
	}
	last := m.Segments[len(m.Segments)-1].URL
	if last != "c.ts" {
		t.Errorf("last segment: got %q, want %q", last, "c.ts")
	}
}
