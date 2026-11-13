package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func isKickUnit(input string) bool {
	return strings.Contains(input, "kick.com") || uuid.Validate(input) == nil
}

func handleUnitInput(
	units *[]interface{},
	in string,
	quality string,
	start, end time.Duration,
	output string,
) {
	var unit interface{}
	if isKickUnit(in) {
		unit = kick.NewUnit(
			in,
			quality,
			kick.WithTimestamps(start, end),
			kick.WithPathname(output),
		)
	} else {
		unit = downloader.NewUnit(
			in,
			downloader.WithQuality(quality),
			downloader.WithTimestamps(start, end),
			downloader.WithPathname(output),
		)
	}
	*units = append(*units, unit)
}

func parseUnitsFile(path string) ([]*Unit, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var units []*Unit
	if err := json.Unmarshal(b, &units); err != nil {
		return nil, err
	}

	return units, nil
}

func sort(units []interface{}) ([]*downloader.Unit, []*kick.Unit) {
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

type ParseUnit struct {
	Inputs     []string
	Quality    string
	Start, End time.Duration
	OutputPath string
}

func ParseUnits(pu ParseUnit) ([]*downloader.Unit, []*kick.Unit, error) {
	units := make([]interface{}, 0)

	for _, input := range pu.Inputs {
		if filepath.Ext(input) != "" {
			inputUnits, err := parseUnitsFile(input)
			if err != nil {
				return nil, nil, err
			}

			for _, unit := range inputUnits {
				if unit.Output == "" {
					unit.Output = pu.OutputPath
				}
				handleUnitInput(&units, unit.Input, unit.Quality, unit.Start, unit.End, unit.Output)
			}
		} else {
			handleUnitInput(&units, input, pu.Quality, pu.Start, pu.End, pu.OutputPath)
		}
	}

	twitch, kick := sort(units)

	return twitch, kick, nil
}
