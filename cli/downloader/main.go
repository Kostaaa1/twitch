package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/prompt"
	"github.com/Kostaaa1/twitch/internal/spinner"
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

	progressCh := make(chan spinner.ChannelMessage, len(units))
	tw.SetProgressChannel(progressCh)

	t := make([]spinner.Unit, len(units))
	for _, u := range units {
		displayPath := ""
		if f, ok := u.W.(*os.File); ok && f != nil {
			displayPath = f.Name()
		}
		t = append(t, spinner.Unit{
			Text:        displayPath,
			TotalBytes:  0,
			ElapsedTime: 0,
			IsDone:      false,
			Err:         u.Error,
		})
	}

	go spinner.New(t, progressCh, jsonCfg.Downloader)

	tw.BatchDownload(units)

	progressCh <- spinner.ChannelMessage{Exit: true}
	close(progressCh)

	time.Sleep(500 * time.Millisecond)
	fmt.Printf("\033[?25h")
}
