package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/gin-gonic/gin"
)

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

func (s *Static) downloadHandler(c *gin.Context) {
	mediaTitle := c.Query("media_title")
	mediaFormat := c.Query("media_format")
	mediaType := c.Query("media_type")

	var unit twitch.DownloadUnit
	unit.Slug = c.Query("media_slug")

	videoType, err := twitch.GetVideoType(mediaType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	unit.Type = videoType
	unit.Quality = mediaFormat
	unit.W = c.Writer

	ext := "mp4"
	if mediaFormat == "audio_only" {
		ext = "mp3"
		c.Header("Content-Type", "audio/mpeg")
	} else {
		c.Header("Content-Type", "video/mp4")
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, mediaTitle, ext))
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	switch unit.Type {
	case twitch.TypeClip:
		if err := s.tw.DownloadClip(unit); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

	case twitch.TypeVOD:
		startH := c.Query("start_h")
		startM := c.Query("start_m")
		startS := c.Query("start_s")
		start, err := parseDuration(startH, startM, startS)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		unit.Start = start

		endH := c.DefaultQuery("end_h", "0")
		endM := c.DefaultQuery("end_m", "0")
		endS := c.DefaultQuery("end_s", "0")
		end, err := parseDuration(endH, endM, endS)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		unit.End = end

		if err := s.tw.StreamVOD(unit); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
}
