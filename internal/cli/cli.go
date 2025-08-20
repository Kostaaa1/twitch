package cli

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
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
			unitsKick = append(unitsKick, kickUnit)
		} else {
			unit := downloader.NewUnit(u.Input, u.Quality, downloader.WithTimestamps(u.Start, u.End))
			unitsTwitch = append(unitsTwitch, *unit)
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
			// path := filepath.Join(opt.Output, fmt.Sprintf("%d.mp4", i))
			// f, err := os.Create(path)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			kickUnit := kick.Unit{URL: input, Quality: downloader.Quality1080p60}
			kickUnits = append(kickUnits, kickUnit)
		} else {
			unit := downloader.NewUnit(input, opt.Quality, downloader.WithTimestamps(opt.Start, opt.End))
			// if withWriter && unit.Error == nil {
			// 	filename, err := dl.MediaTitle(unit.ID, unit.Type)
			// 	if err != nil {
			// 		log.Fatal(err)
			// 	}
			// 	unit.Writer, unit.Error = NewFile(filename, unit.Quality, opt.Output)
			// }
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
