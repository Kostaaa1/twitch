package main

import (
	"log"
	"sync"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/prompt"
	"github.com/Kostaaa1/twitch/internal/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitchdl"
)

func main() {
	jsonCfg, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	dl := twitchdl.New()

	units := prompt.ParseFlags(dl, jsonCfg)
	spin := spinner.New(units, jsonCfg.Downloader.SpinnerModel)
	dl.SetProgressChannel(spin.ProgChan)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		spin.Run()
	}()

	dl.BatchDownload(units)

	wg.Wait()
	close(spin.ProgChan)
}
