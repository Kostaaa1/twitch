package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
	"github.com/Kostaaa1/twitch/pkg/twitch/event"
	"github.com/google/uuid"
)

type Option struct {
	Input   string        `json:"input"`
	Output  string        `json:"output"`
	Quality string        `json:"quality"`
	Start   time.Duration `json:"start"`
	End     time.Duration `json:"end"`

	Set      bool
	Threads  int
	Category string
	Channel  string

	Videos     bool
	Clips      bool
	Highlights bool

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

func NewFile(fileName string, quality downloader.QualityType, output string) (*os.File, error) {
	if output == "" {
		return nil, errors.New("output path not provided")
	}
	ext := "mp4"
	if strings.HasPrefix(quality.String(), "audio") {
		ext = "mp3"
	}
	return fileutil.CreateFile(output, fileName, ext)
}

func EventsFromUnits(tw *twitch.Client, units []downloader.Unit) ([]event.Event, error) {
	var events []event.Event
	for _, unit := range units {
		if unit.Error != nil {
			return nil, unit.Error
		}
		if unit.Type == downloader.TypeLivestream {
			user, err := tw.UserByChannelName(unit.ID)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			events = append(events, event.StreamOnlineEvent(user.ID))
		}
	}
	return events, nil
}

func level(main, fallback *Option) {
	if main.Output == "" && fallback.Output != "" {
		main.Output = fallback.Output
	}
	if main.Quality == "" && fallback.Quality != "" {
		main.Quality = fallback.Quality
	}
}

func isKick(input string) bool {
	return strings.Contains(input, "kick.com") || uuid.Validate(input) == nil
}

func (opt Option) getUnitsFromFileInput(dl *downloader.Downloader, withWriter bool) ([]downloader.Unit, []kick.Unit) {
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

	var twitchUnits []downloader.Unit
	var kickUnits []kick.Unit

	for _, u := range inputUnits {
		level(&u, &opt)

		if isKick(u.Input) {
			kickUnit := kick.Unit{
				URL:     u.Input,
				Quality: downloader.Quality1080p60,
				Start:   u.Start,
				End:     u.End,
			}
			kickUnit.W, kickUnit.Error = NewFile(u.Input, kickUnit.Quality, u.Output)
			kickUnits = append(kickUnits, kickUnit)
		} else {
			unit := downloader.NewUnit(u.Input, u.Quality, downloader.WithTimestamps(u.Start, u.End))
			if unit.Error == nil && withWriter {
				filename, err := dl.MediaTitle(unit.ID, unit.Type)
				if err != nil {
					log.Fatal(err)
				}
				unit.Writer, unit.Error = NewFile(filename, unit.Quality, u.Output)
				twitchUnits = append(twitchUnits, *unit)
				// units[i] = *unit
			}
		}

	}

	return twitchUnits, kickUnits
}

func (opt Option) getUnitsFromFlagInput(dl *downloader.Downloader, withWriter bool) ([]downloader.Unit, []kick.Unit) {
	inputs := strings.Split(opt.Input, ",")

	var twitchUnits []downloader.Unit
	var kickUnits []kick.Unit

	for i, input := range inputs {
		if isKick(input) {
			path := filepath.Join(opt.Output, fmt.Sprintf("%d.mp4", i))
			f, err := os.Create(path)
			if err != nil {
				log.Fatal(err)
			}

			kickUnit := kick.Unit{URL: input, W: f, Quality: downloader.Quality1080p60}
			kickUnits = append(kickUnits, kickUnit)
		} else {
			unit := downloader.NewUnit(input, opt.Quality, downloader.WithTimestamps(opt.Start, opt.End))
			if withWriter && unit.Error == nil {
				filename, err := dl.MediaTitle(unit.ID, unit.Type)
				if err != nil {
					log.Fatal(err)
				}
				unit.Writer, unit.Error = NewFile(filename, unit.Quality, opt.Output)
			}
			twitchUnits = append(twitchUnits, *unit)
		}
	}

	return twitchUnits, kickUnits
}

func (opts Option) UnitsFromInput(dl *downloader.Downloader, createNewUnitWriter bool) ([]downloader.Unit, []kick.Unit) {
	if opts.Input == "" {
		log.Fatalf("Input was not provided.")
	}

	var twitchUnits []downloader.Unit
	var kickUnits []kick.Unit

	_, err := os.Stat(opts.Input)
	if os.IsNotExist(err) {
		twitchUnits, kickUnits = opts.getUnitsFromFlagInput(dl, createNewUnitWriter)
	} else {
		twitchUnits, kickUnits = opts.getUnitsFromFileInput(dl, createNewUnitWriter)
	}

	return twitchUnits, kickUnits
}
