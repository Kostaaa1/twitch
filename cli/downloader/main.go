package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Kostaaa1/twitch/cli/downloader/spinner"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/prompt"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func main() {
	jsonCfg, err := config.Get()
	if err != nil {
		panic(err)
	}

	var prompt prompt.Prompt
	flag.StringVar(&prompt.Input, "input", "", "Provide URL of VOD, clip or livestream to download. You can provide multiple URLs by seperating them with comma. Example: -input=https://www.twitch.tv/videos/2280187162,https://www.twitch.tv/brittt/clip/IronicArtisticOrcaWTRuck-UecXBrM6ECC-DAZR")
	flag.StringVar(&prompt.Output, "output", jsonCfg.Downloader.Output, "Path where to store the downloaded media.")
	flag.StringVar(&prompt.Quality, "quality", "", "[best 1080 720 480 360 160 worst]. Example: -quality 1080p (optional)")
	flag.DurationVar(&prompt.Start, "start", time.Duration(0), "The start of the VOD subset. It only works with VODs and it needs to be in this format: '1h30m0s' (optional)")
	flag.DurationVar(&prompt.End, "end", time.Duration(0), "The end of the VOD subset. It only works with VODs and it needs to be in this format: '1h33m0s' (optional)")
	flag.Parse()

	tw := twitch.New()
	if prompt.Input == "" {
		if len(os.Args) > 1 {
			prompt.Input = os.Args[1]
		}
	}

	units := prompt.ProcessInput(tw)
	progressCh := make(chan twitch.ProgresbarChanData, len(units))
	tw.SetProgressChannel(progressCh)

	go func() {
		paths := make([]string, len(units))
		for i, u := range units {
			if f, ok := u.W.(*os.File); ok {
				paths[i] = f.Name()
			}
		}
		spinner.New(paths, progressCh)
	}()

	if len(units) > 1 {
		tw.BatchDownload(units)
	} else {
		tw.Download(units[0])
	}

	close(progressCh)
	time.Sleep(500 * time.Millisecond)

	// fmt.Printl("Finished downloading 👍👍👍")
	fmt.Printf("\033[?25h")
}
