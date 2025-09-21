package main

import (
	"flag"
	"time"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/internal/config"
)

func ParseFlags(conf config.Config) cli.Option {
	var option cli.Option

	flag.StringVar(&option.Input, "input", "", "input can be twitch (URL, vod id or clip slug), kick (vod URL) or json file (check example.json). Multiple inputs can be comma-separated which will be downloaded concurrently")

	flag.StringVar(&option.Output, "output", conf.Downloader.Output, "Destination path for downloaded files")
	flag.StringVar(&option.Quality, "quality", "", "Video quality: best, 1080, 720, 480, 360, 160, worst, or audio")

	flag.DurationVar(&option.Start, "start", time.Duration(0), "Start time for VOD segment (e.g., 1h30m0s). Only for VODs")
	flag.DurationVar(&option.End, "end", time.Duration(0), "End time for VOD segment (e.g., 1h45m0s). Only for VODs")

	flag.IntVar(&option.Threads, "threads", 10, "Number of parallel downloads (batch mode only)")

	flag.StringVar(&option.Channel, "channel", "", "Twitch channel name")

	flag.BoolVar(&option.Subscribe, "subscribe", false, "Enable live stream monitoring: starts a websocket server and uses channel names from --input flag to automatically download streams when they go live. It could be used in combination with tools such as systemd, to auto-record the stream in the background.")
	flag.BoolVar(&option.Authorize, "auth", false, "Authorize with Twitch. It is mostly needed for CLI chat feature and Helix API. Downloader is not using authorization tokens")

	//

	flag.Parse()

	return option
}
