package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/gin-gonic/gin"
)

func (s *Static) downloadClip(c *gin.Context) {
	mediaTitle := c.Query("media_title")
	mediaFormat := c.Query("media_format")
	slug := c.Query("media_slug")

	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.mp4"`, mediaTitle))

	u := twitch.MediaUnit{
		Slug:    slug,
		Quality: mediaFormat,
		Type:    twitch.TypeClip,
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
		Type:    twitch.TypeVOD,
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
