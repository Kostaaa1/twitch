package cli

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/kick"
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

func (opt Option) unitsFromFileInput() ([]downloader.Unit, []kick.Unit) {
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

	var unitsTwitch []downloader.Unit
	var unitsKick []kick.Unit

	for _, unit := range inputUnits {
		if unit.Output == "" && opt.Output != "" {
			unit.Output = opt.Output
		}
		if unit.Quality == "" && opt.Quality != "" {
			unit.Quality = opt.Quality
		}

		if isKick(unit.Input) {
			kickUnit := kick.Unit{
				URL:     unit.Input,
				Quality: downloader.Quality1080p60,
				Start:   unit.Start,
				End:     unit.End,
			}
			kickUnit.CreateFile(opt.Output)
			unitsKick = append(unitsKick, kickUnit)
		} else {
			twitchUnit := downloader.NewUnit(
				unit.Input,
				unit.Quality,
				downloader.WithTimestamps(unit.Start, unit.End),
				downloader.WithWriter(opt.Output),
			)
			unitsTwitch = append(unitsTwitch, *twitchUnit)
		}
	}

	return unitsTwitch, unitsKick
}

func (opt Option) unitsFromFlagInput() ([]downloader.Unit, []kick.Unit) {
	inputs := strings.Split(opt.Input, ",")

	var twitchUnits []downloader.Unit
	var kickUnits []kick.Unit

	for _, input := range inputs {
		if isKick(input) {
			unit := kick.Unit{
				URL:     input,
				Quality: downloader.Quality1080p60,
				Start:   opt.Start,
				End:     opt.End,
			}
			unit.CreateFile(opt.Output)
			kickUnits = append(kickUnits, unit)
		} else {
			unit := downloader.NewUnit(
				input,
				opt.Quality,
				downloader.WithTimestamps(opt.Start, opt.End),
				downloader.WithWriter(opt.Output),
			)
			unit.Title = uuid.NewString()
			twitchUnits = append(twitchUnits, *unit)
		}
	}

	return twitchUnits, kickUnits
}

func (opts Option) UnitsFromInput() ([]downloader.Unit, []kick.Unit) {
	if opts.Input == "" {
		log.Fatalf("Input was not provided.")
	}

	var twitchUnits []downloader.Unit
	var kickUnits []kick.Unit

	_, err := os.Stat(opts.Input)
	if os.IsNotExist(err) {
		twitchUnits, kickUnits = opts.unitsFromFlagInput()
	} else {
		twitchUnits, kickUnits = opts.unitsFromFileInput()
	}

	return twitchUnits, kickUnits
}
