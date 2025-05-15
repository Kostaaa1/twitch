package options

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitchdl"
)

type Flag struct {
	Input   string        `json:"url"`
	Quality string        `json:"quality"`
	Start   time.Duration `json:"start"`
	End     time.Duration `json:"end"`
	Output  string        `json:"output"`
	// list
	// limit
	Channel   string
	Print     string
	MediaType string
	Limit     int
}

func (p *Flag) UnmarshalJSON(b []byte) error {
	type Alias Flag
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

func level(main, fallback *Flag) {
	if main.Output == "" && fallback.Output != "" {
		main.Output = fallback.Output
	}
	if main.Quality == "" && fallback.Quality != "" {
		main.Quality = fallback.Quality
	}
}

func processFileInput(dl *twitchdl.Downloader, flagOpts Flag) []twitchdl.Unit {
	_, err := os.Stat(flagOpts.Input)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	content, err := os.ReadFile(flagOpts.Input)
	if err != nil {
		log.Fatal(err)
	}

	var opts []Flag
	if err := json.Unmarshal(content, &opts); err != nil {
		log.Fatal(err)
	}

	var units []twitchdl.Unit
	for _, opt := range opts {
		level(&opt, &flagOpts)
		unit := dl.NewUnit(opt.Input, opt.Quality, opt.Output, opt.Start, opt.End)
		units = append(units, unit)
	}

	return units
}

func processFlagInput(dl *twitchdl.Downloader, opt Flag) []twitchdl.Unit {
	urls := strings.Split(opt.Input, ",")
	var units []twitchdl.Unit
	for _, url := range urls {
		opt.Input = url
		unit := dl.NewUnit(url, opt.Quality, opt.Output, opt.Start, opt.End)
		units = append(units, unit)
	}
	return units
}

func GetUnits(dl *twitchdl.Downloader, p Flag) []twitchdl.Unit {
	if p.Input == "" {
		log.Fatalf("Input was not provided.")
	}
	var units []twitchdl.Unit
	_, err := os.Stat(p.Input)
	if os.IsNotExist(err) {
		units = processFlagInput(dl, p)
	} else {
		units = processFileInput(dl, p)
	}
	return units
}
