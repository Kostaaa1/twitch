package main

import (
	"log"
	"sync"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/prompt"
	"github.com/Kostaaa1/twitch/internal/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func main() {
	jsonCfg, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	tw := twitch.New()
	units := prompt.ParseFlags(tw, jsonCfg)

	m := spinner.New(units, jsonCfg.Downloader.SpinnerModel)
	tw.SetProgressChannel(m.ProgChan)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.Run()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tw.BatchDownload(units)
	}()

	wg.Wait()

	close(m.ProgChan)

	// time.Sleep(500 * time.Millisecond)
	// fmt.Printf("\033[?25h")
}
