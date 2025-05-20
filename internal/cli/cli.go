package cli

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
	"github.com/Kostaaa1/twitch/pkg/twitch/event"
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
	Subscribe bool
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

	var inputUnits []Option
	if err := json.Unmarshal(content, &inputUnits); err != nil {
		log.Fatal(err)
	}

	var units []downloader.Unit
	for _, u := range inputUnits {
		level(&u, &opt)
		unit := downloader.NewUnit(u.Input, u.Quality, downloader.WithTimestamps(u.Start, u.End))
		if unit.Error == nil {
			unit.Writer, unit.Error = NewFile(dl, unit, u.Output)
			units = append(units, *unit)
		}
	}

	return units
}

func NewFile(dl *downloader.Downloader, unit *downloader.Unit, output string) (*os.File, error) {
	if output == "" {
		return nil, errors.New("output path not provided")
	}
	fileName, err := dl.MediaTitle(unit.ID, unit.Type)
	if err != nil {
		return nil, err
	}
	ext := "mp4"
	if strings.HasPrefix(unit.Quality.String(), "audio") {
		ext = "mp3"
	}
	return fileutil.CreateFile(output, fileName, ext)
}

func (opt Option) processFlagInput(dl *downloader.Downloader) []downloader.Unit {
	inputs := strings.Split(opt.Input, ",")
	var units []downloader.Unit
	for _, input := range inputs {
		unit := downloader.NewUnit(input, opt.Quality, downloader.WithTimestamps(opt.Start, opt.End))
		if unit.Error == nil {
			unit.Writer, unit.Error = NewFile(dl, unit, opt.Output)
			units = append(units, *unit)
		}
	}
	return units
}

func EventsFromUnits(dl *downloader.Downloader, units []downloader.Unit) ([]event.Event, error) {
	var events []event.Event
	for _, unit := range units {
		if unit.Error != nil {
			return nil, unit.Error
		}
		if unit.Type == downloader.TypeLivestream {
			user, err := dl.TWApi.UserByChannelName(unit.ID)
			if err != nil {
				return nil, err
			}
			events = append(events, event.StreamOnlineEvent(user.ID))
		}
	}
	return events, nil
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
