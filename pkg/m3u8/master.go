package m3u8

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
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

func Master(fetchedPlaylist []byte) *MasterPlaylist {
	master := &MasterPlaylist{
		Serialized: string(fetchedPlaylist),
	}
	master.parse()
	return master
}

func createServingID() string {
	w := strings.Split("0123456789abcdefghijklmnopqrstuvwxyz", "")
	id := ""
	for i := 0; i < 32; i++ {
		id += w[rand.Intn(len(w))]
	}
	return id
}

// Used for sub-only VODs (users need to be subscribed to watch the VOD)
func MasterPlaylistMock(c *http.Client, vodID string, previewURL *url.URL, broadcastType string) *MasterPlaylist {
	master := MasterPlaylist{
		Origin:          "s3",
		B:               false,
		Region:          "EU",
		UserIP:          "127.0.0.1",
		ServingID:       createServingID(),
		Cluster:         "cloudfront_vod",
		UserCountry:     "BE",
		ManifestCluster: "cloudfront_vod",
	}

	paths := strings.Split(previewURL.Path, "/")
	var vodId string
	for i, p := range paths {
		if p == "storyboards" {
			vodId = paths[i-1]
		}
	}

	res := map[string]struct {
		Res string
		FPS string
	}{
		"chunked":    {Res: "1920x1080", FPS: "60"},
		"720p60":     {Res: "1280x720", FPS: "60"},
		"720p30":     {Res: "1280x720", FPS: "30"},
		"480p30":     {Res: "854x480", FPS: "30"},
		"360p30":     {Res: "640x360", FPS: "30"},
		"160p30":     {Res: "284x160", FPS: "30"},
		"audio_only": {Res: "audio_only", FPS: ""},
	}

	isQualityValid := func(u string) bool {
		resp, err := c.Get(u)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}

	for key, value := range res {
		// [NOT TESTED]
		// This method works for older uploaded VODS
		// days_difference - difference between current date and p.Data.Video.CreatedAt
		// if broadcastType == "upload" && days_difference > 7 {
		// url = fmt.Sprintf(`https://${domain}/${channelData.login}/${vodId}/${vodSpecialID}/${resKey}/index-dvr.m3u8`, previewURL.Host, p.Data.Video.Owner.Login, slug, vodId, resolution)
		// }
		// resolution := getResolution(quality, v)

		var URL string
		if strings.ToLower(broadcastType) == "highlight" {
			// https://${domain}/${vodSpecialID}/${resKey}/highlight-${vodId}.m3u8
			URL = fmt.Sprintf(`https://%s/%s/%s/highlight-%s.m3u8`, previewURL.Host, vodId, key, vodID)
			// } else if broadcastType != "upload" {
			// `https://${domain}/${channelData.login}/${vodId}/${vodSpecialID}/${resKey}/index-dvr.m3u8`
			// }
		} else {
			URL = fmt.Sprintf(`https://%s/%s/%s/index-dvr.m3u8`, previewURL.Host, vodId, key)
		}

		// if URL == "" {
		// 	continue
		// }

		if isQualityValid(URL) {
			if key == "chunked" {
				key = "1080p60"
			}
			vp := &VariantPlaylist{
				URL:        URL,
				Bandwidth:  "", // ????
				Codecs:     "avc1.64002A,mp4a.40.2",
				Resolution: value.Res,
				FrameRate:  value.FPS,
				Video:      key,
			}
			master.Lists = append(master.Lists, vp)
		}
	}

	return &master
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

func (playlist *MasterPlaylist) GetVariantPlaylistByQuality(quality string) (*VariantPlaylist, error) {
	for _, list := range playlist.Lists {
		if strings.HasPrefix(list.Video, quality) {
			return list, nil
		}
	}

	if len(playlist.Lists) > 0 {
		return playlist.Lists[0], nil
	}

	return nil, fmt.Errorf("quality not found in master m3u8 playlist: %s", quality)
}
