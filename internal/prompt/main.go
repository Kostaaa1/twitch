package prompt

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/fileutil"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

// type PromptFilter struct {
// 	Day   string `json:"day,24h"`
// 	Week  string `json:"week,7d"`
// 	Month string `json:"month,30d"`
// }

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

func newUnit(tw *twitch.API, prompt Prompt) twitch.MediaUnit {
	// instead of log.Fatal handle it gracefully
	var unit twitch.MediaUnit

	if !twitch.IsQualityValid(prompt.Quality) {
		log.Fatalf("Provided quality is not valid: %s. These are valid qualities: %s", prompt.Quality, strings.Join(twitch.Qualities, ", "))
	}

	slug, vtype, err := tw.Slug(prompt.Input)
	if err != nil {
		log.Fatal(err)
	}

	if vtype == twitch.TypeVOD {
		if prompt.Start > 0 && prompt.End > 0 && prompt.Start >= prompt.End {
			log.Fatalf("invalid time range: Start time (%v) is greater or equal to End time (%v) for URL: %s", prompt.Start, prompt.End, prompt.Input)
		}
	}

	////////////////////////
	// we do not want to do this here, do after trying to get the media, because it can fail. Use unit.SetWriter()
	ext := "mp4"
	if strings.HasPrefix(prompt.Quality, "audio") {
		ext = "mp3"
	}
	mediaName := fmt.Sprintf("%s_%s", slug, prompt.Quality)
	f, err := fileutil.CreateFile(prompt.Output, mediaName, ext)
	if err != nil {
		log.Fatal(err)
	}
	////////////////////////

	unit.Slug = slug
	unit.Type = vtype
	unit.Quality = prompt.Quality
	unit.Start = prompt.Start
	unit.End = prompt.End
	unit.W = f

	return unit
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

	var prompts []Prompt
	if err := json.Unmarshal(content, &prompts); err != nil {
		log.Fatal(err)
	}

	var units []twitch.MediaUnit
	for _, prompt := range prompts {
		units = append(units, newUnit(tw, prompt))
	}

	return units
}

func processFlagInput(tw *twitch.API, prompt Prompt) []twitch.MediaUnit {
	urls := strings.Split(prompt.Input, ",")
	var units []twitch.MediaUnit
	for _, url := range urls {
		prompt.Input = url
		units = append(units, newUnit(tw, prompt))
	}
	return units
}

func (prompt Prompt) ProcessInput(tw *twitch.API) []twitch.MediaUnit {
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
