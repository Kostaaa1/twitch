package cli

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/downloader"
	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/google/uuid"
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
	inputs []string,
	quality string,
	start time.Duration,
	end time.Duration,
	output string,
) ([]*downloader.Unit, []*kick.Unit, error) {
	// units := make([]*downloader.Unit, 0)
	units := make([]interface{}, 0)

	for _, input := range inputs {
		_, err := os.Stat(input)
		if !os.IsNotExist(err) {
			b, err := os.ReadFile(input)
			if err != nil {
				return nil, nil, err
			}

			var inputUnits []*Unit
			if err := json.Unmarshal(b, &inputUnits); err != nil {
				return nil, nil, err
			}

			for _, unit := range inputUnits {
				if unit.Output == "" {
					unit.Output = output
				}

				handleUnitInput(
					&units,
					input,
					quality,
					start,
					end,
					output,
				)
			}
		} else {
			handleUnitInput(
				&units,
				input,
				quality,
				start,
				end,
				output,
			)
		}
	}

	twitch, kick := filterUnits(units)

	return twitch, kick, nil
}

func filterUnits(units []interface{}) ([]*downloader.Unit, []*kick.Unit) {
	twitchUnits := make([]*downloader.Unit, 0)
	kickUnits := make([]*kick.Unit, 0)

	for _, unit := range units {
		if unit, ok := unit.(*downloader.Unit); ok {
			twitchUnits = append(twitchUnits, unit)
			continue
		}
		if unit, ok := unit.(*kick.Unit); ok {
			kickUnits = append(kickUnits, unit)
			continue
		}
		panic("found unit that is not twitch nor kick unit")
	}

	return twitchUnits, kickUnits
}

func isKickUnit(input string) bool {
	return strings.Contains(input, "kick.com") || uuid.Validate(input) == nil
}

func handleUnitInput(
	units *[]interface{},
	input string,
	quality string,
	start, end time.Duration,
	output string,
) {
	var unit interface{}
	if isKickUnit(input) {
		unit = kick.NewUnit(
			input,
			quality,
			kick.WithTimestamps(start, end),
			kick.WithPathname(output),
		)
	} else {
		unit = downloader.NewUnit(
			input,
			downloader.WithQuality(quality),
			downloader.WithTimestamps(start, end),
			downloader.WithPathname(output),
		)
	}

	*units = append(*units, unit)
}
