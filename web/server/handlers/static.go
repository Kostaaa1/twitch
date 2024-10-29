package handlers

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/web/server"
	"github.com/Kostaaa1/twitch/web/views/components"
	"github.com/Kostaaa1/twitch/web/views/home"
	"github.com/gin-gonic/gin"
)

type Static struct {
	tw *twitch.API
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
	re := regexp.MustCompile(`-\d+x\d+\.(jpg|png|jpeg|gif|bmp)$`)
	if match := re.FindStringSubmatch(imgURL); len(match) > 1 {
		ext := match[1]
		newDimensions := fmt.Sprintf("-%dx%d.%s", w, h, ext)
		return re.ReplaceAllString(imgURL, newDimensions)
	}
	return imgURL
}

func (s *Static) GetMediaInfo(c *gin.Context) {
	twitchUrl := c.PostForm("twitchUrl")

	slug, vtype, err := s.tw.Slug(twitchUrl)
	if err != nil {
		fmt.Printf("error while parsing the twitch.tv URL: %v\n", err)
		return
	}

	var formData components.FormData
	var qualities []components.Quality

	if vtype == twitch.TypeVOD {
		metadata, err := s.tw.VideoMetadata(slug)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		master, _, err := s.tw.GetVODMasterM3u8(slug)
		if err != nil {
			return
		}

		for _, list := range master.Lists {
			qualities = append(qualities, components.Quality{
				Resolution: list.Resolution,
				Video:      list.Video,
			})
		}

		formData = components.FormData{
			PreviewThumbnailURL: metadata.Video.PreviewThumbnailURL,
			ID:                  metadata.Video.ID,
			Title:               metadata.Video.Title,
			CreatedAt:           metadata.Video.CreatedAt,
			Owner:               metadata.Video.Owner.DisplayName,
			ViewCount:           metadata.Video.ViewCount,
			Qualities:           qualities,
		}
	}

	if vtype == twitch.TypeClip {
		clip, err := s.tw.ClipData(slug)
		if err != nil {
			fmt.Println(err)
			return
		}

		clipData, _ := s.tw.GetClipData(slug)
		fmt.Println("\nCLIPDATA: ", clipData)

		// for _, quality := range clip.VideoQualities {
		// 	qualities = append(qualities, components.Quality{
		// 		Resolution: quality,
		// 		Video:      list.Video,
		// 	})
		// }

		formData = components.FormData{
			PreviewThumbnailURL: clip.ThumbnailURL,
			ID:                  clip.Slug,
			Title:               clip.Video.Title,
			CreatedAt:           clip.CreatedAt,
			Owner:               clip.Broadcaster.DisplayName,
			ViewCount:           clip.ViewCount,
			Qualities:           qualities,
		}
	}

	resizedImg := replaceImageDimension(formData.PreviewThumbnailURL, 1920, 1080)
	formData.PreviewThumbnailURL = resizedImg
	c.HTML(http.StatusOK, "", server.WithBase(c, components.Form(formData), "Home", ""))
}

func (s *Static) DownloadMedia(c *gin.Context) {
	mediaStart := c.Query("media_start")
	mediaEnd := c.Query("media_end")
	mediaTitle := c.Query("media_title")
	mediaFormat := c.Query("media_format")
	slug := c.Query("media_slug")

	start, err := time.ParseDuration(mediaStart)
	if err != nil {
		start = 0
	}
	end, err := time.ParseDuration(mediaEnd)
	if err != nil {
		end = 0
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

	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.mp4"`, mediaTitle))

	for _, segment := range segments {
		if strings.HasSuffix(segment, ".ts") {
			lastIndex := strings.LastIndex(vodPlaylistURL, "/")
			segmentURL := fmt.Sprintf("%s/%s", vodPlaylistURL[:lastIndex], segment)

			resp, err := http.Get(segmentURL)
			if err != nil {
				fmt.Printf("error fetching segment %s: %v\n", segmentURL, err)
				return
			}
			defer resp.Body.Close()

			_, err = io.Copy(c.Writer, resp.Body)
			if err != nil {
				fmt.Printf("error writing segment %s: %v\n", segmentURL, err)
				return
			}

			if f, ok := c.Writer.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func (s *Static) Register(r *gin.RouterGroup) {
	r.GET("/", s.Root)
	r.GET("/home", s.Home)
	r.POST("/media/info", s.GetMediaInfo)
	r.GET("/media/download", s.DownloadMedia)
}
