package m3u8

import (
	"encoding/json"
	"fmt"
	"strings"
)

func New(fetchedPlaylist []byte) *MasterPlaylist {
	master := &MasterPlaylist{
		Serialized: string(fetchedPlaylist),
	}
	master.Parse()
	return master
}

func (m *MasterPlaylist) Parse() {
	lines := strings.Split(m.Serialized, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			vl := parseVariantPlaylist(line, lines[i+1])
			m.Lists = append(m.Lists, vl)
			i += 2
			if i >= len(lines) {
				break
			}
		}
	}
}

func (playlist *MasterPlaylist) GetVariantPlaylistByQuality(quality string) (VariantPlaylist, error) {
	mediaLists := playlist.Lists
	for i := 0; i < len(mediaLists); i++ {
		list := mediaLists[i]
		if quality == "best" && (list.Video == "1080p60" || list.Video == "chunked") {
			return list, nil
		}
		if strings.HasPrefix(list.Video, quality) {
			return list, nil
		}
	}
	return VariantPlaylist{}, fmt.Errorf("could not find the playlist by provided quality: %s", quality)
}

func (playlist *MasterPlaylist) GetJSONSegments() []string {
	var segments []string
	for _, seg := range playlist.Lists {
		b, err := json.MarshalIndent(seg, "", " ")
		if err != nil {
			break
		}
		segments = append(segments, string(b))
	}
	return segments
}
