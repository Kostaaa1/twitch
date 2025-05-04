package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitchdl"
	"github.com/Kostaaa1/twitch/web/server"
	"github.com/Kostaaa1/twitch/web/views/components"
	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
)

func replaceImageDimension(imgURL string, w, h int) string {
	re := regexp.MustCompile(`-\d+x\d+\.(jpg|png|jpeg|gif|bmp)$`)
	if match := re.FindStringSubmatch(imgURL); len(match) > 1 {
		ext := match[1]
		newDimensions := fmt.Sprintf("-%dx%d.%s", w, h, ext)
		return re.ReplaceAllString(imgURL, newDimensions)
	}
	return imgURL
}

func (s *Static) mediaInfo(c *gin.Context) {
	twitchUrl := c.PostForm("twitchUrl")

	// parsed, err := s.tw.ParseURL(twitchUrl)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	u, err := url.Parse(twitchUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, id := path.Split(u.Path)

	var formData components.FormData
	if strings.Contains(u.Host, "clips.twitch.tv") || strings.Contains(u.Path, "/clip/") {
		formData.Type = twitchdl.TypeClip
		formData, err = s.getClipData(id)
	} else if strings.Contains(u.Path, "/videos/") {
		formData.Type = twitchdl.TypeVOD
		formData, err = s.getVODData(id)
	}

	// if parsed.Type == twitchdl.TypeClip {
	// 	formData, err = s.getClipData(parsed.ID)
	// } else if parsed.Type == twitchdl.TypeVOD {
	// 	formData, err = s.getVODData(parsed.ID)
	// }
	// formData.Type = parsed.Type

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "", server.WithBase(c, components.DownloadForm(formData), "Home", ""))
}

func (s *Static) getVODData(slug string) (components.FormData, error) {
	metadata, err := s.dl.TWApi.VideoMetadata(slug)
	if err != nil {
		return components.FormData{}, err
	}

	master, err := s.dl.TWApi.GetVODMasterM3u8(slug)

	var qualities []components.Quality
	if err != nil {
		return components.FormData{}, err
	}

	for _, list := range master.Lists {
		qualities = append(qualities, components.Quality{
			Resolution: list.Resolution,
			Value:      list.Video,
		})
	}

	duration := time.Duration(metadata.Video.LengthSeconds) * time.Second

	formData := components.FormData{
		PreviewThumbnailURL: replaceImageDimension(metadata.Video.PreviewThumbnailURL, 1920, 1080),
		ID:                  metadata.Video.ID,
		Title:               metadata.Video.Title,
		CreatedAt:           metadata.Video.CreatedAt,
		Owner:               metadata.Video.Owner.DisplayName,
		ViewCount:           humanize.Comma(metadata.Video.ViewCount),
		Qualities:           qualities,
		Duration:            duration.String(),
		Type:                twitchdl.TypeVOD,
	}

	return formData, nil
}

func (s *Static) getClipData(slug string) (components.FormData, error) {
	clip, err := s.dl.TWApi.ClipMetadata(slug)
	if err != nil {
		return components.FormData{}, err
	}

	videoSrc, err := s.dl.GetClipVideoURL(clip, "best")
	if err != nil {
		return components.FormData{}, err
	}

	var qualities []components.Quality
	for _, data := range clip.Assets[0].VideoQualities {
		qualities = append(qualities, components.Quality{
			Resolution: data.Quality,
			Value:      data.Quality,
		})
	}

	qualities = append(qualities, components.Quality{
		Resolution: "audio_only",
		Value:      "audio_only",
	})

	duration := time.Duration(clip.DurationSeconds) * time.Second

	formData := components.FormData{
		PreviewThumbnailURL: clip.Assets[0].ThumbnailURL,
		VideoURL:            videoSrc,
		ID:                  clip.Slug,
		Title:               clip.Title,
		CreatedAt:           clip.CreatedAt,
		Owner:               clip.Broadcaster.DisplayName,
		ViewCount:           humanize.Comma(clip.ViewCount),
		Qualities:           qualities,
		Duration:            duration.String(),
		Curator:             clip.Curator,
		Type:                twitchdl.TypeClip,
	}

	return formData, nil
}
