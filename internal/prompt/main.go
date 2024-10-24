package prompt

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Prompt struct {
	Input   string        `json:"url"`
	Quality string        `json:"quality"`
	Start   time.Duration `json:"start"`
	End     time.Duration `json:"end"`
	Output  string        `json:"output"`
}

func (p *Prompt) UnmarshalJSON(b []byte) error {
	type Alias Prompt
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

func processFileInput(tw *twitch.Client, input string) []twitch.MediaUnit {
	content, err := os.ReadFile(input)
	if err != nil {
		panic(err)
	}

	var body []Prompt
	if err := json.Unmarshal(content, &body); err != nil {
		panic(err)
	}

	var units []twitch.MediaUnit

	for _, b := range body {
		unit, err := tw.NewMediaUnit(b.Input, b.Quality, b.Output, b.Start, b.End)
		if err != nil {
			panic(err)
		}
		units = append(units, unit)
	}
	return units
}

func processFlagInput(tw *twitch.Client, prompt *Prompt) []twitch.MediaUnit {
	urls := strings.Split(prompt.Input, ",")

	var units []twitch.MediaUnit

	for _, url := range urls {
		unit, err := tw.NewMediaUnit(url, prompt.Quality, prompt.Output, prompt.Start, prompt.End)
		if err != nil {
			panic(err)
		}
		units = append(units, unit)
	}

	return units
}

func (prompt *Prompt) ProcessInput(tw *twitch.Client) []twitch.MediaUnit {
	if prompt.Input == "" {
		log.Fatalf("Input was not provided.")
	}

	var units []twitch.MediaUnit

	_, err := os.Stat(prompt.Input)
	if os.IsNotExist(err) {
		units = processFlagInput(tw, prompt)
	} else {
		units = processFileInput(tw, prompt.Input)
	}
	return units
}
