package prompt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
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

func createNewUnit(tw *twitch.API, prompt Prompt) twitch.MediaUnit {
	var unit twitch.MediaUnit

	slug, vtype, err := tw.Slug(prompt.Input)
	if err != nil {
		unit.Error = err
	}

	if vtype == twitch.TypeVOD {
		if prompt.Start > 0 && prompt.End > 0 && prompt.Start >= prompt.End {
			unit.Error = fmt.Errorf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL: %s", prompt.Start, prompt.End, prompt.Input)
		}
	}

	quality, err := twitch.GetQuality(prompt.Quality, vtype)
	if err != nil {
		unit.Error = err
	}

	var f *os.File
	ext := "mp4"
	if quality == "audio_only" {
		ext = "mp3"
	}

	mediaName := fmt.Sprintf("%s_%s", slug, quality)
	f, err = fileutil.CreateFile(prompt.Output, mediaName, ext)
	if err != nil {
		unit.Error = err
	}

	unit.Slug = slug
	unit.Type = vtype
	unit.Quality = quality
	unit.Start = prompt.Start
	unit.End = prompt.End
	unit.W = f

	return unit
}

func processFileInput(tw *twitch.API, input string) []twitch.MediaUnit {
	_, err := os.Stat(input)
	if os.IsNotExist(err) {
		panic(err)
	}

	content, err := os.ReadFile(input)
	if err != nil {
		panic(err)
	}

	var prompts []Prompt
	if err := json.Unmarshal(content, &prompts); err != nil {
		panic(err)
	}

	var units []twitch.MediaUnit
	for _, prompt := range prompts {
		units = append(units, createNewUnit(tw, prompt))
	}

	return units
}

func processFlagInput(tw *twitch.API, prompt Prompt) []twitch.MediaUnit {
	urls := strings.Split(prompt.Input, ",")
	var units []twitch.MediaUnit
	for _, url := range urls {
		prompt.Input = url
		units = append(units, createNewUnit(tw, prompt))
	}
	return units
}

func (prompt Prompt) ProcessInput(tw *twitch.API) []twitch.MediaUnit {
	if prompt.Input == "" {
		panic("Input was not provided.")
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
