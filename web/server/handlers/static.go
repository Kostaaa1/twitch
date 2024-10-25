package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/web/server"
	"github.com/Kostaaa1/twitch/web/views/components"
	"github.com/Kostaaa1/twitch/web/views/home"
	"github.com/gin-gonic/gin"
)

type Static struct {
	tw *twitch.Client
}

func NewStatic() *Static {
	return &Static{
		tw: twitch.New(),
	}
}

func (*Static) Root(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/home")
}

func (*Static) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "", server.WithBase(c, home.Home(), "Home", "homepage"))
}

func replaceImageDimension(imgURL string, w, h int) string {
	lastDashIndex := strings.LastIndex(imgURL, "-")
	if lastDashIndex == -1 {
		return imgURL
	}

	xIndex := strings.Index(imgURL[lastDashIndex:], "x")
	if xIndex == -1 {
		return imgURL
	}
	xIndex += lastDashIndex

	base := imgURL[:lastDashIndex+1]
	newDimensions := fmt.Sprintf("%d%s%d.jpg", w, "x", h)

	return base + newDimensions
}

func (s *Static) GetMediaInfo(c *gin.Context) {
	twitchUrl := c.PostForm("twitchUrl")
	slug, vtype, err := s.tw.Slug(twitchUrl)
	if err != nil {
		return
	}

	if vtype == twitch.TypeVOD {
		metadata, err := s.tw.VideoMetadata(slug)
		if err != nil {
		}
		master, _, err := s.tw.GetVODMasterM3u8(slug)
		if err != nil {
		}

		resizedImg := replaceImageDimension(metadata.Video.PreviewThumbnailURL, 1920, 1080)
		formData := components.FormData{
			Title:               metadata.Video.Title,
			Slug:                slug,
			VariantLists:        master.Lists,
			PreviewThumbnailURL: resizedImg,
			ViewCount:           metadata.Video.ViewCount,
			LengthSeconds:       metadata.Video.LengthSeconds,
		}

		c.HTML(http.StatusOK, "", server.WithBase(c, components.Form(formData), "Home", ""))
	}
}

func (s *Static) DownloadMedia(c *gin.Context) {
	media_start := c.Query("media_start")
	media_end := c.Query("media_end")
	mediaFormat := c.Query("media_format")
	slug := c.Query("media_slug")

	start, err := time.ParseDuration(media_start)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid media_start")
		return
	}
	end, err := time.ParseDuration(media_end)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid media_end")
		return
	}

	vodPlaylistURL, err := s.tw.GetVODMediaPlaylist(slug, mediaFormat)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get VOD playlist URL")
		return
	}

	resp, err := http.Get(vodPlaylistURL)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch playlist")
		return
	}
	defer resp.Body.Close()

	mediaPlaylist, err := io.ReadAll(resp.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read playlist data")
		return
	}

	segments := s.tw.GetSegments(mediaPlaylist, start, end)

	c.Header("Content-Type", "video/mp2t")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.mp4"`, slug))

	for _, tsFile := range segments {
		lastIndex := strings.LastIndex(vodPlaylistURL, "/")
		segmentURL := fmt.Sprintf("%s/%s", vodPlaylistURL[:lastIndex], tsFile)

		segmentResp, err := http.Get(segmentURL)
		if err != nil {
			fmt.Println("Error fetching segment:", err)
			continue
		}
		defer segmentResp.Body.Close()

		if _, err := io.Copy(c.Writer, segmentResp.Body); err != nil {
			fmt.Println("Error while writing bytes to temp file:", err)
			break
		}
	}
}

func (s *Static) Register(r *gin.RouterGroup) {
	r.GET("/", s.Root)
	r.GET("/home", s.Home)
	r.POST("/media/info", s.GetMediaInfo)
	r.GET("/media/download", s.DownloadMedia)
}
