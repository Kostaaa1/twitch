package downloader

import (
	"testing"

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
	testCases := []testCase{
		{
			input:      "https://www.twitch.tv/stableronaldo",
			expectedID: "stableronaldo",
			// inputQuality:    "best",
			// expectedQuality: Quality1080p60,
		},
		{
			input:      "https://www.twitch.tv/mizkif/clip/BombasticAgitatedGazelleCoolStoryBob-6eoUuzyD75YH7Ycc?filter=clips&range=7d&sort=time",
			expectedID: "BombasticAgitatedGazelleCoolStoryBob-6eoUuzyD75YH7Ycc",
		},
		{
			input:      "https://www.twitch.tv/videos/2587289805?filter=archives&sort=time",
			expectedID: "2587289805",
		},
		{
			input:       "https://www.twitch.tv/videos/2587289805?t=6h50m30s",
			expectedID:  "2587289805",
			expectStart: true,
		},
	}

	for _, tc := range testCases {
		t.Parallel()
		var unit *Unit

		switch {
		case tc.inputQuality != "":
			unit = NewUnit(tc.input, WithQuality(tc.inputQuality))
		default:
			unit = NewUnit(tc.input)
		}

		require.NoError(t, unit.Error)
		require.Equal(t, unit.ID, tc.expectedID)
		require.Equal(t, unit.Type, tc)
		require.Equal(t, unit.Quality, tc.expectedQuality)
		require.Equal(t, unit.Type, tc.expectedType)

		if tc.expectStart {
			require.NotZero(t, unit.Start)
		} else {
			require.Zero(t, unit.Start)
		}
	}
}
