package m3u8

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed testdata/master_vod.m3u8
var masterVOD string

//go:embed testdata/master_live.m3u8
var masterLive string

// parseMaster is a thin shim so the test bodies describe behavior, not the
// exact parser signature. Update this one helper when the API moves to
// `ParseMaster(io.Reader) (*MasterPlaylist, error)`.
func parseMaster(t *testing.T, s string) *MasterPlaylist {
	t.Helper()
	return Master([]byte(s))
}

func TestParseMaster_VOD_TwitchInfoFields(t *testing.T) {
	m := parseMaster(t, masterVOD)

	cases := map[string]string{
		"Origin":          "s3",
		"Region":          "EU",
		"UserIP":          "203.0.113.42",
		"ServingID":       "abcdef0123456789abcdef0123456789",
		"Cluster":         "cloudfront_vod",
		"UserCountry":     "US",
		"ManifestCluster": "cloudfront_vod",
	}

	got := map[string]string{
		"Origin":          m.Origin,
		"Region":          m.Region,
		"UserIP":          m.UserIP,
		"ServingID":       m.ServingID,
		"Cluster":         m.Cluster,
		"UserCountry":     m.UserCountry,
		"ManifestCluster": m.ManifestCluster,
	}

	for k, want := range cases {
		if got[k] != want {
			t.Errorf("%s: got %q, want %q", k, got[k], want)
		}
	}

	if m.B != false {
		t.Errorf("B: got %v, want false", m.B)
	}
}

// Catches the bug at master.go:154 where USER-COUNTRY is assigned to Cluster.
func TestParseMaster_UserCountryDoesNotOverwriteCluster(t *testing.T) {
	m := parseMaster(t, masterVOD)

	if m.Cluster != "cloudfront_vod" {
		t.Errorf("Cluster was overwritten by USER-COUNTRY: got %q", m.Cluster)
	}
	if m.UserCountry != "US" {
		t.Errorf("UserCountry: got %q, want %q", m.UserCountry, "US")
	}
}

func TestParseMaster_VOD_AllVariantsParsed(t *testing.T) {
	m := parseMaster(t, masterVOD)

	if len(m.Lists) != 7 {
		t.Fatalf("expected 7 variants in VOD master, got %d", len(m.Lists))
	}

	wantVideos := []string{"1080p60", "720p60", "720p30", "480p30", "360p30", "160p30", "audio_only"}
	for i, want := range wantVideos {
		if m.Lists[i].Video != want {
			t.Errorf("variant[%d].Video: got %q, want %q", i, m.Lists[i].Video, want)
		}
	}
}

func TestParseMaster_Live_AllVariantsParsed(t *testing.T) {
	m := parseMaster(t, masterLive)

	if len(m.Lists) != 3 {
		t.Fatalf("expected 3 variants in live master, got %d", len(m.Lists))
	}
	wantVideos := []string{"1080p60", "720p60", "audio_only"}
	for i, want := range wantVideos {
		if m.Lists[i].Video != want {
			t.Errorf("variant[%d].Video: got %q, want %q", i, m.Lists[i].Video, want)
		}
	}
}

// Catches the bug at variant.go:23 where splitting by "," corrupts
// CODECS="avc1.64002A,mp4a.40.2".
func TestParseMaster_CodecsWithEmbeddedComma(t *testing.T) {
	m := parseMaster(t, masterVOD)

	if len(m.Lists) == 0 {
		t.Fatal("no variants parsed")
	}
	want := "avc1.64002A,mp4a.40.2"
	if m.Lists[0].Codecs != want {
		t.Errorf("Codecs: got %q, want %q", m.Lists[0].Codecs, want)
	}
}

func TestParseMaster_VariantFields(t *testing.T) {
	m := parseMaster(t, masterVOD)

	if len(m.Lists) < 2 {
		t.Fatalf("expected at least 2 variants, got %d", len(m.Lists))
	}

	v := m.Lists[0]
	if v.Bandwidth != "8534030" {
		t.Errorf("Bandwidth: got %q, want %q", v.Bandwidth, "8534030")
	}
	if v.Resolution != "1920x1080" {
		t.Errorf("Resolution: got %q, want %q", v.Resolution, "1920x1080")
	}
	if v.FrameRate != "60.000" {
		t.Errorf("FrameRate: got %q, want %q", v.FrameRate, "60.000")
	}
	if v.URL != "https://d2nvs31859zcd8.cloudfront.net/abcdef0123/chunked/index-dvr.m3u8" {
		t.Errorf("URL: got %q", v.URL)
	}
}

// Catches the bug at master.go:172 where `i += 2` followed by the for-loop's
// own `i++` skips an entire line. Two consecutive STREAM-INF blocks with no
// #EXT-X-MEDIA between them expose it. The real-world fixtures don't trigger
// it because Twitch always emits #EXT-X-MEDIA between variants — that just
// means the bug is masked, not absent.
func TestParseMaster_DoesNotSkipConsecutiveStreamInf(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=1,RESOLUTION=1x1,VIDEO="a"
https://example.com/a.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2,RESOLUTION=2x2,VIDEO="b"
https://example.com/b.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=3,RESOLUTION=3x3,VIDEO="c"
https://example.com/c.m3u8
`
	m := parseMaster(t, playlist)

	if len(m.Lists) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(m.Lists))
	}
	wantURLs := []string{
		"https://example.com/a.m3u8",
		"https://example.com/b.m3u8",
		"https://example.com/c.m3u8",
	}
	for i, want := range wantURLs {
		if m.Lists[i].URL != want {
			t.Errorf("variant[%d].URL: got %q, want %q", i, m.Lists[i].URL, want)
		}
	}
}

func TestParseMaster_ChunkedRemappedTo1080p60(t *testing.T) {
	m := parseMaster(t, masterVOD)
	if len(m.Lists) == 0 {
		t.Fatal("no variants")
	}
	if m.Lists[0].Video != "1080p60" {
		t.Errorf("expected `chunked` to be remapped to 1080p60, got %q", m.Lists[0].Video)
	}
}

func TestParseMaster_AudioOnlyResolutionLabel(t *testing.T) {
	m := parseMaster(t, masterVOD)
	var audio *VariantPlaylist
	for _, v := range m.Lists {
		if v.Video == "audio_only" {
			audio = v
		}
	}
	if audio == nil {
		t.Fatal("audio_only variant not found")
	}
	if audio.Resolution != "Audio only" {
		t.Errorf("Resolution: got %q, want %q", audio.Resolution, "Audio only")
	}
}

// Real-world manifests use CRLF.
func TestParseMaster_HandlesCRLFLineEndings(t *testing.T) {
	playlist := strings.ReplaceAll(masterVOD, "\n", "\r\n")
	m := parseMaster(t, playlist)
	if len(m.Lists) != 7 {
		t.Errorf("CRLF playlist: got %d variants, want 7", len(m.Lists))
	}
	if m.Origin != "s3" {
		t.Errorf("CRLF playlist: Origin not parsed, got %q", m.Origin)
	}
}

func TestGetVariantPlaylistByQuality(t *testing.T) {
	m := parseMaster(t, masterVOD)

	tests := []struct {
		quality   string
		wantVideo string
	}{
		{"1080", "1080p60"},
		{"720", "720p60"},
		{"480", "480p30"},
		{"audio", "audio_only"},
	}

	for _, tt := range tests {
		t.Run(tt.quality, func(t *testing.T) {
			v, err := m.VariantPlaylistByQuality(tt.quality)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Video != tt.wantVideo {
				t.Errorf("got %q, want %q", v.Video, tt.wantVideo)
			}
		})
	}
}

func TestGetVariantPlaylistByQuality_UnknownFallsBackToBest(t *testing.T) {
	m := parseMaster(t, masterVOD)
	v, err := m.VariantPlaylistByQuality("ultra-hd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Video != "1080p60" {
		t.Errorf("expected fallback to first variant (1080p60), got %q", v.Video)
	}
}

// Garbage input must not panic and must not produce phantom variants.
func TestParseMaster_RejectsNonM3UInput(t *testing.T) {
	t.Skip("API improvement: Master should return an error for input lacking #EXTM3U")

	m := parseMaster(t, "this is not a playlist\nrandom garbage\n")
	if m == nil {
		return
	}
	if len(m.Lists) != 0 {
		t.Errorf("got %d phantom variants from garbage input", len(m.Lists))
	}
}

func TestParseMaster_EmptyInputDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on empty input: %v", r)
		}
	}()
	_ = parseMaster(t, "")
}

// Twitch master manifests carry #EXT-X-MEDIA blocks; the parser ought to at
// least not lose the GROUP-ID/NAME info. Skipped today; remove the Skip when
// you add #EXT-X-MEDIA handling.
func TestParseMaster_CapturesMediaGroupNames(t *testing.T) {
	t.Skip("feature: parse #EXT-X-MEDIA into a MediaGroups field on MasterPlaylist")
}
