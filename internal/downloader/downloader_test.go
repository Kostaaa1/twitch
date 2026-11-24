package downloader

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSegmentDownload_ExtractSegmentExtension(t *testing.T) {
	cases := []struct {
		have string
		want string
	}{
		{
			have: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18.mp4",
			want: "",
		},
		{
			have: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/23-unmuted.mp4",
			want: "-unmuted",
		},
		{
			have: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/46-muted.mp4",
			want: "-muted",
		},
	}
	for _, tc := range cases {
		ttransformed := segmentSuffix(tc.have)
		require.Equal(t, ttransformed, tc.want)
	}
}

func TestSegmentDownload_StripSegmentURLSuffix(t *testing.T) {
	cases := []struct {
		have string
		t    string
		want string
	}{
		{
			have: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/init-0.mp4",
			t:    "-muted",
			want: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/init-0.mp4",
		},
		{
			have: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18.mp4",
			t:    "-unmuted",
			want: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18-unmuted.mp4",
		},
		{
			have: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/23-unmuted.mp4",
			t:    "",
			want: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/23.mp4",
		},
		{
			have: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/46.mp4",
			t:    "",
			want: "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/46.mp4",
		},
	}

	for _, tc := range cases {
		ttransformed := withSegmentSuffix(tc.have, tc.t)
		require.Equal(t, ttransformed, tc.want)
	}
}

// func TestSegmentDownload_TransformSegmentURL(t *testing.T) {
// 	cases := []struct {
// 		have     string
// 		want     string
// 		wantCond bool
// 	}{
// 		{
// 			have:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/init-0.mp4",
// 			want:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/init-0.mp4",
// 			wantCond: true,
// 		},
// 		{
// 			have:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18.mp4",
// 			want:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18-unmuted.mp4",
// 			wantCond: false,
// 		},
// 		{
// 			have:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18-unmuted.mp4",
// 			want:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18-muted.mp4",
// 			wantCond: false,
// 		},
// 		{
// 			have:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18-muted.mp4",
// 			want:     "https://d2vi6trrdongqn.cloudfront.net/4a6540bcd7dda31826e7_channel_319485162848_1783892695/1080p60/18-muted.mp4",
// 			wantCond: true,
// 		},
// 	}

// 	for _, tc := range cases {
// 		transformed, cond := transformSegmentURL(tc.have)
// 		require.Equal(t, transformed, tc.want)
// 		require.Equal(t, cond, tc.wantCond)
// 	}
// }
