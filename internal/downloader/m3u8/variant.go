package m3u8

import (
	"strings"
)

type VariantPlaylist struct {
	Bandwidth  string
	Codecs     string
	Resolution string
	Video      string
	FrameRate  string
	URL        string
	Serialized string
}

func parseVariantPlaylist(line, URL string) *VariantPlaylist {
	var variant VariantPlaylist
	variant.URL = URL

	line = strings.TrimPrefix(line, "#EXT-X-STREAM-INF:")
	params := strings.Split(line, ",")

	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "BANDWIDTH":
			variant.Bandwidth = value
		case "CODECS":
			variant.Codecs = strings.Trim(value, `"`)
		case "RESOLUTION":
			variant.Resolution = value
		case "VIDEO":
			v := strings.Trim(value, `"`)
			variant.Video = v
			if v == "chunked" {
				variant.Video = "1080p60"
			} else if v == "audio_only" {
				variant.Resolution = "Audio only"
			}

		case "FRAME-RATE":
			variant.FrameRate = value
		}
	}

	return &variant
}
