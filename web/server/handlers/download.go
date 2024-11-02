package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

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
	fmt.Println("called handler")

	twitchUrl := c.PostForm("twitchUrl")
	slug, vtype, err := s.tw.Slug(twitchUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch vtype {
	case twitch.TypeClip:
		formData, err := s.getClipData(slug)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.HTML(http.StatusOK, "", server.WithBase(c, components.DownloadClipForm(formData), "Home", ""))

	case twitch.TypeVOD:
		formData, err := s.getVODData(slug)
		fmt.Println(formData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.HTML(http.StatusOK, "", server.WithBase(c, components.DownloadVODForm(formData), "Home", ""))

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported media type"})
	}
}

func (s *Static) getVODData(slug string) (components.FormData, error) {
	metadata, err := s.tw.VideoMetadata(slug)
	if err != nil {
		return components.FormData{}, err
	}

	fmt.Println("\n DATA: ", metadata)

	master, _, err := s.tw.GetVODMasterM3u8(slug)
	if err != nil {
		return components.FormData{}, nil
	}

	var qualities []components.Quality
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
	}

	return formData, nil
}

func (s *Static) getClipData(slug string) (components.FormData, error) {
	clip, err := s.tw.ClipData(slug)
	if err != nil {
		return components.FormData{}, err
	}

	var qualities []components.Quality
	for _, q := range clip.Assets[0].VideoQualities {
		res := twitch.GetResolution(q.Quality, twitch.TypeClip)
		// WORK ON THIS:
		qualities = append(qualities, components.Quality{
			Resolution: res,
			Value:      res,
		})
	}

	formData := components.FormData{
		PreviewThumbnailURL: clip.Assets[0].ThumbnailURL,
		ID:                  clip.Slug,
		Title:               clip.Video.Title,
		CreatedAt:           clip.CreatedAt,
		Owner:               clip.Broadcaster.DisplayName,
		ViewCount:           humanize.Comma(clip.ViewCount),
		Qualities:           qualities,
	}

	return formData, nil
}

func (s *Static) downloadClip(c *gin.Context) {
	mediaTitle := c.Query("media_title")
	mediaFormat := c.Query("media_format")
	slug := c.Query("media_slug")

	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.mp4"`, mediaTitle))

	u := twitch.MediaUnit{
		Slug:    slug,
		Quality: mediaFormat,
		Vtype:   twitch.TypeClip,
		W:       c.Writer,
	}

	if err := s.tw.DownloadClip(u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func parseDuration(startH, startM, startS string) (time.Duration, error) {
	hours, err := strconv.Atoi(startH)
	if err != nil || startH == "" {
		hours = 0
	}
	minutes, err := strconv.Atoi(startM)
	if err != nil || startM == "" {
		minutes = 0
	}
	seconds, err := strconv.Atoi(startS)
	if err != nil || startS == "" {
		seconds = 0
	}

	duration := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	return duration, nil
}

func (s *Static) downloadVOD(c *gin.Context) {
	mediaTitle := c.Query("media_title")
	mediaFormat := c.Query("media_format")
	slug := c.Query("media_slug")

	startH := c.Query("start_h")
	startM := c.Query("start_m")
	startS := c.Query("start_s")

	start, err := parseDuration(startH, startM, startS)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endH := c.DefaultQuery("end_h", "0")
	endM := c.DefaultQuery("end_m", "0")
	endS := c.DefaultQuery("end_s", "0")

	end, err := parseDuration(endH, endM, endS)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	unit := twitch.MediaUnit{
		Slug:    slug,
		Vtype:   twitch.TypeVOD,
		Start:   start,
		End:     end,
		Quality: mediaFormat,
		W:       c.Writer,
	}

	ext := "mp4"
	if mediaFormat == "audio_only" {
		ext = "mp3"
		c.Header("Content-Type", "audio/mpeg")
	} else {
		c.Header("Content-Type", "video/mp4")
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, mediaTitle, ext))

	if err := s.tw.StreamVOD(unit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}
