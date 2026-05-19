package twitch

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMasterPlaylistFromMedia(t *testing.T) {
	// https://www.twitch.tv/videos/2436889062?filter=highlights&sort=time

	var testcases = map[string]struct {
		name string
		id   string
		want string
	}{
		"highlight": {
			id:   "2727569143",
			name: "highlight playlist",
			want: "https://d2vi6trrdongqn.cloudfront.net/e2e07ed4312999528e01_stableronaldo_29966191484_7471102211/chunked/highlight-2727569143.m3u8",
		},
		"upload": {
			id:   "1199379108",
			name: "upload playlist",
			want: "https://d2nvs31859zcd8.cloudfront.net/xqcow/1199379108/75f3517e-2c89-4ea3-b6f7-ad9a57b053e1/720p60/index-dvr.m3u8",
		},
		"vod": {
			id:   "2770593719",
			name: "vod playlist",
		},
		"sub_vod": {
			id:   "2769452309",
			name: "sub vod playlist",
		},
	}

	twc := NewClient()
	ctx := context.Background()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			pup, err := twc.MasterURL(ctx, tc.id)
			require.NoError(t, err)
			require.Equal(t, pup, tc.want)
			require.True(t, reflect.DeepEqual(pup, tc.want))
		})
	}
}
