package cmd

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kostaaa1/twitch/internal/web/server"
	"github.com/Kostaaa1/twitch/internal/web/server/handlers"
	"github.com/spf13/cobra"
)

var (
	logLevel string
	port     int
)

func parseLevel(s string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(s))
	return level, err
}

func runWebServer(args []string) error {
	level, err := parseLevel(logLevel)
	if err != nil {
		slog.Error("failed to parse log level", slog.Any("err", err), slog.String("log-level", logLevel))
		os.Exit(1)
	}

	slog.SetLogLoggerLevel(level)

	server, err := server.NewServer(port, handlers.NewStatic())
	if err != nil {
		slog.Error("failed to create server", slog.Any("err", err))
		os.Exit(1)
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)

		slog.Info("starting server with port", slog.Int("port", port))

		if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", slog.Any("err", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("Shutting down gracefully...")
	case <-stopped:
		slog.Error("Server stopped unexpectedly, shutting down...")
	}

	if err := server.Stop(10 * time.Second); err != nil {
		slog.Error("Server failed to shutdown gracefully", slog.Any("err", err))
		os.Exit(1)
	}

	return nil
}

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// if len(args) == 0 {
		// 	log.Fatal("invalid usage: info <channel_name>")
		// }
		// if err := runWebServer(args); err != nil {
		// 	log.Fatal(err)
		// }
	},
}

func init() {
	// rootCmd.AddCommand(webCmd)
	// webCmd.PersistentFlags().StringVarP(&logLevel, "level", "l", "", "")
	// webCmd.PersistentFlags().IntVarP(&port, "port", "p", 3000, "")
}
