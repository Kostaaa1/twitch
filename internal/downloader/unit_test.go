package downloader

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testCase struct {
	input           string
	inputQuality    string
	expectedID      string
	expectedQuality QualityType
	expectedType    MediaType
	expectStart     bool
}

func TestUnits(t *testing.T) {
	t.Parallel()

	testCases := []testCase{
		{
			input:           "https://www.twitch.tv/stableronaldo",
			inputQuality:    "",
			expectedID:      "stableronaldo",
			expectedType:    TypeLivestream,
			expectedQuality: Quality1080p60,
		},
		{
			input:        "https://www.twitch.tv/mizkif/clip/BombasticAgitatedGazelleCoolStoryBob-6eoUuzyD75YH7Ycc?filter=clips&range=7d&sort=time",
			expectedID:   "BombasticAgitatedGazelleCoolStoryBob-6eoUuzyD75YH7Ycc",
			expectedType: TypeClip,
		},
		{
			input:           "https://www.twitch.tv/videos/2587289805?filter=archives&sort=time",
			expectedID:      "2587289805",
			expectedType:    TypeVOD,
			inputQuality:    "160p60",
			expectedQuality: Quality160p30,
		},
		{
			input:        "https://www.twitch.tv/videos/2587289805?t=6h50m30s",
			expectedID:   "2587289805",
			expectStart:  true,
			expectedType: TypeVOD,
		},
	}

	for _, tc := range testCases {
		var unit *Unit

		switch {
		case tc.inputQuality != "":
			unit = NewUnit(tc.input, WithQuality(tc.inputQuality))
		default:
			unit = NewUnit(tc.input)
		}

		require.NoError(t, unit.Error)
		require.Equal(t, unit.ID, tc.expectedID)
		require.Equal(t, unit.Type, tc.expectedType)

		if tc.inputQuality != "" {
			require.Equal(t, unit.Quality, tc.expectedQuality)
		} else {
			require.Equal(t, unit.Quality, Quality1080p60)
		}

		if tc.expectStart {
			require.NotZero(t, unit.Start)
		} else {
			require.Zero(t, unit.Start)
		}
	}
}

// Template tests. The unit gets its metadata after the GQL fetch; here it is
// set directly. ParseTemplate returns a relative path without extension.
//
// Placeholders: {channel} {clipper} {name} {id} {quality} {date} {time}
// {date} -> 2006-01-02, {time} -> 15-04-05 (no colons, they're not allowed on Windows)

// newTemplateUnit builds a unit the way the CLI does and attaches metadata,
// as if it had already been fetched.
func newTemplateUnit(t *testing.T, meta Metadata) *Unit {
	t.Helper()
	u := NewUnit(meta.ID)
	require.NoError(t, u.Error)
	u.metadata = &meta
	return u
}

var (
	vodMeta = Metadata{
		Channel: "xqc",
		Name:    "GTA RP day 3",
		ID:      "2926188923",
		Quality: "1080p60",
		Date:    time.Date(2026, 11, 23, 20, 15, 0, 0, time.UTC),
	}
	clipMeta = Metadata{
		Channel: "xqc",
		Clipper: "forsen",
		Name:    "insane play",
		ID:      "DeliciousDelightfulPicklesWOOP",
		Quality: "1080p60",
		Date:    time.Date(2026, 11, 20, 9, 30, 0, 0, time.UTC),
	}
	streamMeta = Metadata{
		Channel: "xqc",
		Name:    "GTA RP day 4",
		ID:      "xqc",
		Quality: "720p60",
		Date:    time.Date(2026, 11, 27, 18, 4, 11, 0, time.UTC),
	}
)

func TestTemplate(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		data Metadata
		want string // slash-separated, compared with filepath.FromSlash
	}{
		// defaults
		{"video default", "{date}_{name}_{id}", vodMeta, "2026-11-23_GTA RP day 3_2926188923"},
		{"clip default", "{channel}/clips/{date}_{name}_{id}", clipMeta, "xqc/clips/2026-11-20_insane play_DeliciousDelightfulPicklesWOOP"},
		{"stream default", "{channel}/live/{date}_{time}", streamMeta, "xqc/live/2026-11-27_18-04-11"},
		// variants
		{"folder per channel", "{channel}/{date}_{name}_{id}", vodMeta, "xqc/2026-11-23_GTA RP day 3_2926188923"},
		{"folder per day", "{channel}/{date}/{name}_{id}", vodMeta, "xqc/2026-11-23/GTA RP day 3_2926188923"},
		{"id only", "{id}", vodMeta, "2926188923"},
		{"with quality", "{channel}_{name}_{quality}_{id}", vodMeta, "xqc_GTA RP day 3_1080p60_2926188923"},
		{"fixed subfolder", "archive/{channel}/{date}_{id}", vodMeta, "archive/xqc/2026-11-23_2926188923"},
		{"clip with clipper", "{channel}/clips/{clipper}_{name}_{id}", clipMeta, "xqc/clips/forsen_insane play_DeliciousDelightfulPicklesWOOP"},
		{"clip by clipper and quality", "{clipper}/{quality}_{id}", clipMeta, "forsen/1080p60_DeliciousDelightfulPicklesWOOP"},
		{"stream with quality", "{channel}/live/{date}_{time}_{quality}", streamMeta, "xqc/live/2026-11-27_18-04-11_720p60"},
		{"placeholder used twice", "{id}/{id}", vodMeta, "2926188923/2926188923"},
		{"no placeholders", "video", vodMeta, "video"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTemplateUnit(t, tt.data).ParseTemplate(tt.tmpl)
			require.NoError(t, err)
			require.Equal(t, filepath.FromSlash(tt.want), got)
		})
	}
}

func TestTemplate_SanitizesValues(t *testing.T) {
	with := func(d Metadata, edit func(*Metadata)) Metadata {
		edit(&d)
		return d
	}

	tests := []struct {
		name string
		tmpl string
		data Metadata
		want string
	}{
		{"slash in title doesn't create a folder", "{name}_{id}",
			with(vodMeta, func(d *Metadata) { d.Name = "AC/DC reaction" }),
			"AC_DC reaction_2926188923"},
		{"backslash in title", "{name}_{id}",
			with(vodMeta, func(d *Metadata) { d.Name = `a\b` }),
			"a_b_2926188923"},
		{"reserved characters", "{name}",
			with(vodMeta, func(d *Metadata) { d.Name = `what?? 🦑🦑` }),
			"what__ 🦑🦑"},
		{"all reserved characters", "{name}",
			with(vodMeta, func(d *Metadata) { d.Name = `a:b*c"d<e>f|g` }),
			"a_b_c_d_e_f_g"},
		{"control characters", "{name}",
			with(vodMeta, func(d *Metadata) { d.Name = "line1\nline2\ttab" }),
			"line1_line2_tab"},
		{"surrounding spaces trimmed", "{name}_{id}",
			with(vodMeta, func(d *Metadata) { d.Name = "   spaced   " }),
			"spaced_2926188923"},
		{"trailing dots trimmed", "{name}_{id}",
			with(vodMeta, func(d *Metadata) { d.Name = "wait..." }),
			"wait_2926188923"},
		{"empty title falls back to id", "{name}_{id}",
			with(vodMeta, func(d *Metadata) { d.Name = "" }),
			"2926188923_2926188923"},
		{"whitespace-only title falls back to id", "{name}",
			with(vodMeta, func(d *Metadata) { d.Name = "   " }),
			"2926188923"},
		{"dots-only title falls back to id", "{name}",
			with(vodMeta, func(d *Metadata) { d.Name = ".." }),
			"2926188923"},
		{"slash in clipper", "{clipper}/{id}",
			with(clipMeta, func(d *Metadata) { d.Clipper = "a/b" }),
			"a_b/DeliciousDelightfulPicklesWOOP"},
		{"long title cut to 60 bytes, id kept", "{name}_{id}",
			with(vodMeta, func(d *Metadata) { d.Name = strings.Repeat("a", 300) }),
			strings.Repeat("a", 60) + "_2926188923"},
		{"cut never splits a multi-byte character", "{name}",
			// 1 + 20*4 = 81 bytes; 15 squids would be 61 bytes, so only 14 fit
			with(vodMeta, func(d *Metadata) { d.Name = "a" + strings.Repeat("🦑", 20) }),
			"a" + strings.Repeat("🦑", 14)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTemplateUnit(t, tt.data).ParseTemplate(tt.tmpl)
			require.NoError(t, err)
			require.Equal(t, filepath.FromSlash(tt.want), got)
		})
	}
}

func TestTemplate_Rejects(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
	}{
		{"empty", ""},
		{"only spaces", "   "},
		{"parent dir", "../{name}"},
		{"parent dir in the middle", "{channel}/../../{id}"},
		{"absolute path", "/home/me/{name}"},
		{"ends with extension mp4", "{name}.mp4"},
		{"ends with extension ts", "{id}.ts"},
		{"no filename, only a folder", "{channel}/"},
		{"empty path segment", "{channel}//{id}"},
		{"unknown placeholder", "{title}_{id}"},
		{"unclosed brace", "{name_{id}"},
		{"stray closing brace", "name}_{id}"},
		{"empty placeholder", "{}_{id}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTemplateUnit(t, vodMeta).ParseTemplate(tt.tmpl)
			require.Error(t, err)
		})
	}
}

func TestTemplate_MissingMetadata(t *testing.T) {
	u := NewUnit(vodMeta.ID)
	require.NoError(t, u.Error)

	_, err := u.ParseTemplate("{date}_{name}_{id}")
	require.Error(t, err)
}
