package cli

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
	"github.com/google/uuid"
)

type Category int

const (
	Latest Category = iota
	MostPopular
)

type Option struct {
	Input   string        `json:"input"`
	Output  string        `json:"output"`
	Quality string        `json:"quality"`
	Start   time.Duration `json:"start"`
	End     time.Duration `json:"end"`

	Threads    int
	Category   string
	Channel    string
	Videos     bool
	Clips      bool
	Highlights bool
	Authorize  bool
	Subscribe  bool
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

func isKick(input string) bool {
	return strings.Contains(input, "kick.com") || uuid.Validate(input) == nil
}

func (opt Option) unitsFromFileInput(units *[]spinner.UnitProvider) {
	_, err := os.Stat(opt.Input)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	content, err := os.ReadFile(opt.Input)
	if err != nil {
		log.Fatal(err)
	}

	var inputUnits []Option
	if err := json.Unmarshal(content, &inputUnits); err != nil {
		log.Fatal(err)
	}

	for _, unit := range inputUnits {
		if unit.Output == "" && opt.Output != "" {
			unit.Output = opt.Output
		}
		if unit.Quality == "" && opt.Quality != "" {
			unit.Quality = opt.Quality
		}

		if isKick(unit.Input) {
			unit := kick.NewUnit(
				unit.Input,
				unit.Quality,
				kick.WithTimestamps(opt.Start, opt.End),
				kick.WithWriter(opt.Output),
			)
			*units = append(*units, unit)
		} else {
			unit := downloader.NewUnit(
				unit.Input,
				unit.Quality,
				downloader.WithTimestamps(unit.Start, unit.End),
				downloader.WithWriter(opt.Output),
			)
			*units = append(*units, unit)
		}
	}

}

func (opt Option) unitsFromFlagInput(units *[]spinner.UnitProvider) {
	inputs := strings.Split(opt.Input, ",")

	for _, input := range inputs {
		if isKick(input) {
			unit := kick.NewUnit(
				input,
				opt.Quality,
				kick.WithTimestamps(opt.Start, opt.End),
				kick.WithWriter(opt.Output),
			)
			*units = append(*units, unit)
		} else {
			unit := downloader.NewUnit(
				input,
				opt.Quality,
				downloader.WithTimestamps(opt.Start, opt.End),
				downloader.WithWriter(opt.Output),
			)
			*units = append(*units, unit)
		}
	}
}

func (opts Option) UnitsFromInput() []spinner.UnitProvider {
	if opts.Input == "" {
		log.Fatalf("Input was not provided.")
	}

	var units []spinner.UnitProvider

	_, err := os.Stat(opts.Input)
	if os.IsNotExist(err) {
		opts.unitsFromFlagInput(&units)
	} else {
		opts.unitsFromFileInput(&units)
	}

	return units
}

func FilterUnits(units []spinner.UnitProvider) ([]downloader.Unit, []kick.Unit) {
	var twitchUnits []downloader.Unit
	var kickUnits []kick.Unit

	for _, unit := range units {
		switch u := unit.(type) {
		case *downloader.Unit:
			twitchUnits = append(twitchUnits, *u)
		case *kick.Unit:
			kickUnits = append(kickUnits, *u)
		}
	}

	return twitchUnits, kickUnits
}
