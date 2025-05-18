package cli

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
)

type Option struct {
	Input     string        `json:"input"`
	Output    string        `json:"output"`
	Quality   string        `json:"quality"`
	Start     time.Duration `json:"start"`
	End       time.Duration `json:"end"`
	Threads   int
	Category  string
	Channel   string
	Authorize bool
	Subscribe string
}

func (p *Option) UnmarshalJSON(b []byte) error {
	type Alias Option
	aux := &struct {
		Start string `json:"start"`
		End   string `json:"end"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	var err error
	if aux.Start != "" {
		p.Start, err = time.ParseDuration(aux.Start)
		if err != nil {
			return err
		}
	}

	if aux.End != "" {
		p.End, err = time.ParseDuration(aux.End)
		if err != nil {
			return err
		}
	}

	return nil
}

func level(main, fallback *Option) {
	if main.Output == "" && fallback.Output != "" {
		main.Output = fallback.Output
	}
	if main.Quality == "" && fallback.Quality != "" {
		main.Quality = fallback.Quality
	}
}

func (opt Option) processFileInput(dl *downloader.Downloader) []downloader.Unit {
	_, err := os.Stat(opt.Input)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	content, err := os.ReadFile(opt.Input)
	if err != nil {
		log.Fatal(err)
	}

	var opts []Option
	if err := json.Unmarshal(content, &opts); err != nil {
		log.Fatal(err)
	}

	var units []downloader.Unit
	for _, unitOpt := range opts {
		level(&unitOpt, &opt)
		unit := dl.NewUnit(unitOpt.Input, unitOpt.Quality, unitOpt.Output, unitOpt.Start, unitOpt.End)
		units = append(units, unit)
	}

	return units
}

func (opt Option) processFlagInput(dl *downloader.Downloader) []downloader.Unit {
	urls := strings.Split(opt.Input, ",")
	var units []downloader.Unit
	for _, url := range urls {
		opt.Input = url
		unit := dl.NewUnit(url, opt.Quality, opt.Output, opt.Start, opt.End)
		units = append(units, unit)
	}
	return units
}

func (opts Option) ProcessFlags(dl *downloader.Downloader) []downloader.Unit {
	if opts.Input == "" {
		log.Fatalf("Input was not provided.")
	}
	var units []downloader.Unit
	_, err := os.Stat(opts.Input)
	if os.IsNotExist(err) {
		units = opts.processFlagInput(dl)
	} else {
		units = opts.processFileInput(dl)
	}
	return units
}
