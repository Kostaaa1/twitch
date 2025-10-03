package cli

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
	"github.com/google/uuid"
)

type Category int

const (
	Latest Category = iota
	MostPopular
)

type Flag struct {
	Input      string        `json:"input"`
	Output     string        `json:"output"`
	Quality    string        `json:"quality"`
	Start      time.Duration `json:"start"`
	End        time.Duration `json:"end"`
	Threads    int
	Category   string
	Info       string
	Videos     bool
	Clips      bool
	Highlights bool
	Authorize  bool
	Subscribe  bool
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

func ParseFlags(conf config.Config) Flag {
	var f Flag

	flag.StringVar(&f.Input, "i", "", "input can be twitch (URL, vod id or clip slug), kick (vod URL) or json file (check example.json). Multiple inputs can be comma-separated which will be downloaded concurrently")
	flag.StringVar(&f.Output, "o", conf.Downloader.Output, "Destination path for downloaded files")
	flag.StringVar(&f.Quality, "q", "", "Video quality: best, 1080, 720, 480, 360, 160, worst, or audio")
	flag.DurationVar(&f.Start, "s", time.Duration(0), "Start time for VOD segment (e.g., 1h30m0s). Only for VODs")
	flag.DurationVar(&f.End, "e", time.Duration(0), "End time for VOD segment (e.g., 1h45m0s). Only for VODs")
	flag.StringVar(&f.Info, "info", "", "channel/vod id/slug for printing in JSON format")
	flag.IntVar(&f.Threads, "threads", 10, "Number of parallel downloads (batch mode only)")
	flag.BoolVar(&f.Subscribe, "subscribe", false, "Enable live stream monitoring: starts a websocket server and uses channel names from --input flag to automatically download streams when they go live. It could be used in combination with tools such as systemd, to auto-record the stream in the background.")
	flag.BoolVar(&f.Authorize, "auth", false, "Authorize with Twitch. It is mostly needed for CLI chat feature and Helix API. Downloader is not using authorization tokens")

	flag.Parse()

	return f
}

func isKick(input string) bool {
	return strings.Contains(input, "kick.com") || uuid.Validate(input) == nil
}

func (flag Flag) unitsFromFlagInput(units *[]spinner.UnitProvider) {
	inputs := strings.Split(flag.Input, ",")

	c := twitch.NewClient(nil)

	for _, input := range inputs {
		if isKick(input) {
			*units = append(*units, kick.NewUnit(
				input,
				flag.Quality,
				kick.WithTimestamps(flag.Start, flag.End),
				kick.WithWriter(flag.Output),
			))
		} else {
			*units = append(*units, downloader.NewUnit(
				input,
				downloader.WithTitle(c),
				downloader.WithQuality(flag.Quality),
				downloader.WithTimestamps(flag.Start, flag.End),
				downloader.WithWriter(flag.Output),
			))
		}
	}
}

func (flag Flag) unitsFromFileInput(units *[]spinner.UnitProvider) {
	_, err := os.Stat(flag.Input)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	content, err := os.ReadFile(flag.Input)
	if err != nil {
		log.Fatal(err)
	}

	var inputUnits []Flag
	if err := json.Unmarshal(content, &inputUnits); err != nil {
		log.Fatal(err)
	}

	c := twitch.NewClient(nil)

	for _, unit := range inputUnits {
		if unit.Output == "" && flag.Output != "" {
			unit.Output = flag.Output
		}
		if unit.Quality == "" && flag.Quality != "" {
			unit.Quality = flag.Quality
		}

		if isKick(unit.Input) {
			*units = append(*units, kick.NewUnit(
				unit.Input,
				unit.Quality,
				kick.WithTimestamps(flag.Start, flag.End),
				kick.WithWriter(flag.Output),
			))
		} else {
			*units = append(*units, downloader.NewUnit(
				unit.Input,
				downloader.WithTitle(c),
				downloader.WithQuality(unit.Quality),
				downloader.WithTimestamps(unit.Start, unit.End),
				downloader.WithWriter(flag.Output),
			))
		}
	}
}

func (opts Flag) UnitsFromInput() []spinner.UnitProvider {
	if opts.Input == "" {
		log.Fatalf("Input was not provided.")
	}

	units := []spinner.UnitProvider{}

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
