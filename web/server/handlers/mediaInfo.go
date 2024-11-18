package handlers

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/Kostaaa1/twitch/pkg/twitch"
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
	slug, vtype, err := s.tw.Slug(twitchUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var formData components.FormData
	if vtype == twitch.TypeClip {
		formData, err = s.getClipData(slug)
	} else if vtype == twitch.TypeVOD {
		formData, err = s.getVODData(slug)
	}
	formData.Type = vtype

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.HTML(http.StatusOK, "", server.WithBase(c, components.DownloadForm(formData), "Home", ""))
}

func (s *Static) getVODData(slug string) (components.FormData, error) {
	metadata, err := s.tw.VideoMetadata(slug)
	if err != nil {
		return components.FormData{}, err
	}

	master, _, err := s.tw.GetVODMasterM3u8(slug)

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

	formData := components.FormData{
		PreviewThumbnailURL: replaceImageDimension(metadata.Video.PreviewThumbnailURL, 1920, 1080),
		ID:                  metadata.Video.ID,
		Title:               metadata.Video.Title,
		CreatedAt:           metadata.Video.CreatedAt,
		Owner:               metadata.Video.Owner.DisplayName,
		ViewCount:           humanize.Comma(metadata.Video.ViewCount),
		Qualities:           qualities,
		MediaDuration:       fmt.Sprintf("%.2f", float64(metadata.Video.LengthSeconds)/3600.00),
	}

	return formData, nil
}

func (s *Static) getClipData(slug string) (components.FormData, error) {
	clip, err := s.tw.ClipData(slug)
	if err != nil {
		return components.FormData{}, err
	}

	videoSrc, err := s.tw.GetClipVideoURL(clip, "best")
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

	formData := components.FormData{
		PreviewThumbnailURL: clip.Assets[0].ThumbnailURL,
		VideoURL:            videoSrc,
		ID:                  clip.Slug,
		Title:               clip.Title,
		CreatedAt:           clip.CreatedAt,
		Owner:               clip.Broadcaster.DisplayName,
		ViewCount:           humanize.Comma(clip.ViewCount),
		Qualities:           qualities,
		MediaDuration:       fmt.Sprintf("%.2f", float64(clip.DurationSeconds)),
		Curator:             clip.Curator,
	}

	return formData, nil
}
