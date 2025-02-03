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

	units := prompt.ParseFlags(jsonCfg)

	m := spinner.New(units, jsonCfg.Downloader.SpinnerModel)

	dl := twitchdl.New()
	dl.SetProgressChannel(m.ProgChan)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.Run()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		dl.BatchDownload(units)
	}()
	wg.Wait()

	close(m.ProgChan)
}
