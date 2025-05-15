package main

import (
	"flag"
	"time"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/options"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitchdl"
)

var (
	option options.Flag
	conf   *config.Config
)

func init() {
	var err error
	conf, err = config.Get()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&option.Input, "input", "", "Provide URL of VOD, clip or livestream to download. You can provide multiple URLs by seperating them by comma. Example: -input=https://www.twitch.tv/videos/2280187162,https://www.twitch.tv/brittt/clip/IronicArtisticOrcaWTRuck-UecXBrM6ECC-DAZR")
	flag.StringVar(&option.Output, "output", conf.Downloader.Output, "Downloaded media path.")
	flag.StringVar(&option.Quality, "quality", "", "[best|1080|720|480|360|160|worst|audio]")
	flag.DurationVar(&option.Start, "start", time.Duration(0), "The start of the VOD subset. It only works with VODs and it needs to be in this format: '1h30m0s' (optional)")
	flag.DurationVar(&option.End, "end", time.Duration(0), "The end of the VOD subset. It only works with VODs and it needs to be in format: '1h33m0s' (optional)")

	flag.StringVar(&option.Channel, "channel", "", "Twitch channel name")
	flag.StringVar(&option.Print, "print", "", "print videos information")
	flag.StringVar(&option.Print, "type", "", "vod|clip|highlight")
	flag.IntVar(&option.Limit, "limit", 25, "limit of the list records (default: 25)")

	// twitchdl --channel mizkif --print videos --limit 25
	// twitchdl --channel mizkif --limit 25 // download latest 25 videos?

	flag.Parse()
}

func main() {
	// if option.Channel != "" {
	// 	videos, err := dl.TWApi.GetVideosByChannelName(option.Channel, option.Limit)
	// 	if err != nil {
	// 		log.Fatal("failed to get the videos by username: %w", err)
	// 		return
	// 	}
	// 	// if option.Print != "" {
	// 	for _, video := range videos {
	// 		u := fmt.Sprintf("[%s | %s] - %s", video.Game.Name, video.ID, video.Title)
	// 		fmt.Println(u)
	// 	}
	// 	// }
	// 	return
	// }

	client := twitch.NewClient(nil, &conf.Creds)
	dl := twitchdl.New(client, conf.Downloader)
	units := options.GetUnits(dl, option)

	// spin := spinner.New(units, conf.Downloader.SpinnerModel)
	// dl.SetProgressChannel(spin.ProgChan)
	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	spin.Run()
	// }()

	dl.BatchDownload(units)

	// wg.Wait()
	// close(spin.ProgChan)
}
