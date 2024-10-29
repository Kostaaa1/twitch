package prompt

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

// type PromptFilter struct {
// 	Day   string `json:"day,24h"`
// 	Week  string `json:"week,7d"`
// 	Month string `json:"month,30d"`
// }

type Prompt struct {
	Input   string        `json:"input,url"`
	Quality string        `json:"quality"`
	Start   time.Duration `json:"start"`
	End     time.Duration `json:"end"`
	Output  string        `json:"destpath,dstpath,output"`
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

func processFileInput(tw *twitch.API, input string) []twitch.MediaUnit {
	_, err := os.Stat(input)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	content, err := os.ReadFile(input)
	if err != nil {
		log.Fatal(err)
	}

	var body []Prompt
	if err := json.Unmarshal(content, &body); err != nil {
		log.Fatal(err)
	}

	var units []twitch.MediaUnit

	for _, b := range body {
		unit, err := tw.NewMediaUnit(b.Input, b.Quality, b.Output, b.Start, b.End)
		if err != nil {
			log.Fatal(err)
		}
		units = append(units, unit)
	}

	return units
}

func processFlagInput(tw *twitch.API, prompt *Prompt) []twitch.MediaUnit {
	urls := strings.Split(prompt.Input, ",")
	var units []twitch.MediaUnit

	for _, url := range urls {
		unit, err := tw.NewMediaUnit(url, prompt.Quality, prompt.Output, prompt.Start, prompt.End)
		if err != nil {
			log.Fatal(err)
		}
		units = append(units, unit)
	}

	return units
}

func (prompt *Prompt) ProcessInput(tw *twitch.API) []twitch.MediaUnit {
	if prompt.Input == "" {
		log.Fatalf("Input was not provided.")
	}

	var units []twitch.MediaUnit

	_, err := url.ParseRequestURI(prompt.Input)

	if err == nil {
		units = processFlagInput(tw, prompt)
	} else {
		units = processFileInput(tw, prompt.Input)
	}

	return units
}
