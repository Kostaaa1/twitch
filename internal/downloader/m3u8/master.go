package m3u8

import (
	"fmt"
	"strconv"
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
	Lists           []*VariantPlaylist
	Serialized      string
}

// TODO: ALL PARSING MUST GO THROUGH READER
func Master(fetchedPlaylist []byte) *MasterPlaylist {
	master := &MasterPlaylist{Serialized: string(fetchedPlaylist)}
	master.parse()
	return master
}

func (m *MasterPlaylist) parseLineInfo(line string) {
	t := strings.Split(line, ":")[1]
	kv := strings.Split(t, ",")

	for _, pair := range kv {
		parts := strings.Split(pair, "=")
		key := parts[0]
		value, err := strconv.Unquote(parts[1])
		if err != nil {
			continue
		}

		switch key {
		case "ORIGIN":
			m.Origin = value
		case "B":
			b, _ := strconv.ParseBool(value)
			m.B = b
		case "REGION":
			m.Region = value
		case "USER-IP":
			m.UserIP = value
		case "SERVING-ID":
			m.ServingID = value
		case "CLUSTER":
			m.Cluster = value
		case "USER-COUNTRY":
			m.Cluster = value
		case "MANIFEST-CLUSTER":
			m.ManifestCluster = value
		}
	}
}

func (m *MasterPlaylist) parse() {
	lines := strings.Split(m.Serialized, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "#EXT-X-TWITCH-INFO:") {
			m.parseLineInfo(line)
		}

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

func (playlist *MasterPlaylist) VariantPlaylistByQuality(quality string) (*VariantPlaylist, error) {
	for _, list := range playlist.Lists {
		if strings.HasPrefix(list.Video, quality) {
			return list, nil
		}
	}
	if len(playlist.Lists) > 0 {
		return playlist.Lists[0], nil
	}
	return nil, fmt.Errorf("error: quality not found in master.m3u8: %s", quality)
}
