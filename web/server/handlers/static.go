package handlers

import (
	"net/http"

	"github.com/Kostaaa1/twitch/web/server"
	"github.com/Kostaaa1/twitch/web/views/home"
	"github.com/gin-gonic/gin"
)

type Static struct {
}

func NewStatic() *Static {
	return &Static{}
}

func (*Static) Root(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/home")
}

func (*Static) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "", server.WithBase(c, home.Home(), "Home", "homepage"))
}

func (*Static) GetMediaInfo(c *gin.Context) {
	twitchUrl := c.PostForm("twitchUrl")
	c.HTML(http.StatusOK, "", server.WithBase(c, home.DownloadForm(twitchUrl), "Home", ""))
}

func (s *Static) Register(r *gin.RouterGroup) {
	r.GET("/", s.Root)
	r.GET("/home", s.Home)
	r.POST("/getMediaInfo", s.GetMediaInfo)
}
