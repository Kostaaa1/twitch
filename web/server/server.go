package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Kostaaa1/twitch/web/server/middleware"
	"github.com/Kostaaa1/twitch/web/views/assets"
	"github.com/Kostaaa1/twitch/web/views/error_pages"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

type Server struct {
	server *http.Server
}

type Handler interface {
	Register(*gin.RouterGroup)
}

func NewServer(addr string, handler ...Handler) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	g := gin.New()
	g.Use(middleware.Logger, gin.Recovery(), middleware.AssetsCache, gzip.Gzip(gzip.DefaultCompression))
	g.HTMLRender = &templRenderer{}

	g.StaticFS("/assets", http.FS(assets.Assets))

	g.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "", WithBase(c, error_pages.NotFound(), "Not found", ""))
	})

	rg := g.Group("/")
	for _, h := range handler {
		h.Register(rg)
	}

	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: g,
			// ReadTimeout:    30 * time.Second,
			// WriteTimeout:   30 * time.Second,
			// IdleTimeout:    120 * time.Second,
			// MaxHeaderBytes: 1 << 20,
		},
	}, nil
}

func (s *Server) Run() error {
	return s.server.ListenAndServe()
}

func (s *Server) Stop(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.server.Shutdown(ctx)
}
