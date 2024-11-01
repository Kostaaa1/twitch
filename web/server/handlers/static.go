package handlers

import (
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/web/server"
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

func (s *Static) Register(r *gin.RouterGroup) {
	r.GET("/", s.Root)
	r.GET("/home", s.Home)
	r.POST("/media/info", s.mediaInfo)
	r.GET("/media/download/vod", s.downloadVOD)
	r.GET("/media/download/clip", s.downloadClip)
}
