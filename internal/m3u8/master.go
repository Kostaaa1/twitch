package m3u8

import (
	"fmt"
	"strings"
)

type MasterPlaylist struct {
	Origin          string `m3u8:"ORIGIN"`
	B               bool   `m3u8:"B"`
	Region          string `m3u8:"REGION"`
	UserIP          string `m3u8:"USER-IP"`
	ServingID       string `m3u8:"SERVING-ID"`
	Cluster         string `m3u8:"CLUSTER"`
	UserCountry     string `m3u8:"USER-COUNTRY"`
	ManifestCluster string `m3u8:"MANIFEST-CLUSTER"`
	UsherURL        string
	Lists           []VariantPlaylist
	Serialized      string
}

func Master(fetchedPlaylist []byte) *MasterPlaylist {
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
	if quality == "best" || quality == "chunked" {
		return mediaLists[0], nil
	}
	if quality == "worst" {
		return mediaLists[len(mediaLists)-1], nil
	}
	for _, list := range mediaLists {
		if strings.HasPrefix(list.Video, quality) {
			return list, nil
		}
	}
	return VariantPlaylist{}, fmt.Errorf("could not find the playlist by provided quality: %s", quality)
}
