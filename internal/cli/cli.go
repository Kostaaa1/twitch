package cli

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Kostaaa1/twitch/internal/downloader"
)

type Unit struct {
	Input   string        `json:"input"`
	Output  string        `json:"output"`
	Quality string        `json:"quality"`
	Start   time.Duration `json:"start"`
	End     time.Duration `json:"end"`
}

func (p *Unit) UnmarshalJSON(b []byte) error {
	type Alias Unit
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

func ParseUnits(
	args []string,
	quality string,
	start time.Duration,
	end time.Duration,
	output string,
) ([]*downloader.Unit, error) {
	units := make([]*downloader.Unit, 0)

	for _, input := range args {
		_, err := os.Stat(input)
		if !os.IsNotExist(err) {
			b, err := os.ReadFile(input)
			if err != nil {
				return nil, err
			}

			var inputUnits []*Unit
			if err := json.Unmarshal(b, &inputUnits); err != nil {
				return nil, err
			}

			for _, unit := range inputUnits {
				if unit.Output == "" {
					unit.Output = output
				}

				unit := downloader.NewUnit(unit.Input,
					downloader.WithQuality(unit.Quality),
					downloader.WithTimestamps(unit.Start, unit.End),
					downloader.WithPathname(unit.Output),
				)
				units = append(units, unit)
			}
		} else {
			unit := downloader.NewUnit(
				input,
				downloader.WithQuality(quality),
				downloader.WithTimestamps(start, end),
				downloader.WithPathname(output),
			)
			units = append(units, unit)
		}
	}

	return units, nil
}

// func isKickEndpoint(input string) bool {
// 	return strings.Contains(input, "kick.com") || uuid.Validate(input) == nil
// }

// func (flag Flag) unitsFromFlagInput(ctx context.Context, c *twitch.Client, units *[]spinner.UnitProvider, ch chan<- spinner.Message) {
// 	inputs := strings.Split(flag.Input, ",")

// 	for _, input := range inputs {
// 		if isKickEndpoint(input) {
// 			*units = append(*units, kick.NewUnit(
// 				input,
// 				flag.Quality,
// 				kick.WithTimestamps(flag.Start, flag.End),
// 			))
// 		} else {
// 			// *units = append(*units, downloader.NewUnit(
// 			// 	input,
// 			// 	downloader.WithQuality(flag.Quality),
// 			// 	downloader.WithTimestamps(flag.Start, flag.End),
// 			// 	downloader.WithFile(ctx, c, flag.Output),
// 			// ))

// 			unit := downloader.NewUnit(
// 				input,
// 				downloader.WithQuality(flag.Quality),
// 				downloader.WithTimestamps(flag.Start, flag.End),
// 				downloader.WithFile(ctx, c, flag.Output),
// 			)
// 			msg := spinner.Message{ID: unit.ID}
// 			ch <- msg
// 			*units = append(*units, unit)
// 		}
// 	}
// }

// func (flag Flag) unitsFromFileInput(ctx context.Context, tw *twitch.Client, units *[]spinner.UnitProvider, ch chan<- spinner.Message) error {
// 	_, err := os.Stat(flag.Input)
// 	if os.IsNotExist(err) {
// 		return err
// 	}

// 	content, err := os.ReadFile(flag.Input)
// 	if err != nil {
// 		return err
// 	}

// 	var inputUnits []Flag
// 	if err := json.Unmarshal(content, &inputUnits); err != nil {
// 		return err
// 	}

// 	for _, unit := range inputUnits {
// 		if unit.Output == "" && flag.Output != "" {
// 			unit.Output = flag.Output
// 		}
// 		if unit.Quality == "" && flag.Quality != "" {
// 			unit.Quality = flag.Quality
// 		}

// 		if isKickEndpoint(unit.Input) {
// 			// *units = append(*units, kick.NewUnit(
// 			// 	unit.Input,
// 			// 	unit.Quality,
// 			// 	kick.WithTimestamps(unit.Start, unit.End),
// 			// 	kick.WithWriter(unit.Output),
// 			// ))
// 		} else {
// 			unit := downloader.NewUnit(
// 				unit.Input,
// 				downloader.WithQuality(unit.Quality),
// 				downloader.WithTimestamps(unit.Start, unit.End),
// 				downloader.WithFile(ctx, tw, unit.Output),
// 			)
// 			msg := spinner.Message{ID: unit.GetID()}
// 			ch <- msg
// 			*units = append(*units, unit)
// 		}
// 	}

// 	return nil
// }

// func (opts Flag) UnitsFromInput(ctx context.Context, tw *twitch.Client, ch chan<- spinner.Message) ([]spinner.UnitProvider, error) {
// 	if opts.Input == "" {
// 		return nil, errors.New("missing input")
// 	}

// 	units := make([]spinner.UnitProvider, 0)

// 	_, err := os.Stat(opts.Input)
// 	if os.IsNotExist(err) {
// 		opts.unitsFromFlagInput(ctx, tw, &units, ch)
// 	} else {
// 		opts.unitsFromFileInput(ctx, tw, &units, ch)
// 	}

// 	return units, nil
// }

// func FilterUnits(units []spinner.UnitProvider) ([]downloader.Unit, []kick.Unit) {
// 	var twitchUnits []downloader.Unit
// 	var kickUnits []kick.Unit

// 	for _, unit := range units {
// 		switch u := unit.(type) {
// 		case *downloader.Unit:
// 			twitchUnits = append(twitchUnits, *u)
// 		case *kick.Unit:
// 			kickUnits = append(kickUnits, *u)
// 		}
// 	}

// 	return twitchUnits, kickUnits
// }
