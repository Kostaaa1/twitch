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
	m := spinner.New(units, jsonCfg.Downloader.SpinnerModel)
	dl.SetProgressChannel(m.ProgChan)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.Run()
	}()

	dl.BatchDownload(units)

	wg.Wait()
	close(m.ProgChan)
}
