package main

import (
	"fmt"
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

	tw := twitch.New()

	prompt := prompt.ParseFlags(jsonCfg)
	units := prompt.ProcessInput(tw)

	progressCh := make(chan twitch.ProgresbarChanData, len(units))
	tw.SetProgressChannel(progressCh)

	go func() {
		spinner.New(units, progressCh, jsonCfg.Downloader)
	}()

	tw.BatchDownload(units)

	progressCh <- twitch.ProgresbarChanData{Exit: true}
	close(progressCh)

	time.Sleep(500 * time.Millisecond)
	fmt.Printf("\033[?25h")
}
