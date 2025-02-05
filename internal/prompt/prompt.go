package prompt

import (
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitchdl"
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

func levelPropmts(main, fallback *Prompt) {
	if main.Output == "" && fallback.Output != "" {
		main.Output = fallback.Output
	}
	if main.Quality == "" && fallback.Quality != "" {
		main.Quality = fallback.Quality
	}
}

func processFileInput(dl *twitchdl.Downloader, inputPrompt Prompt) []twitchdl.DownloadUnit {
	_, err := os.Stat(inputPrompt.Input)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	content, err := os.ReadFile(inputPrompt.Input)
	if err != nil {
		log.Fatal(err)
	}

	var prompts []Prompt
	if err := json.Unmarshal(content, &prompts); err != nil {
		log.Fatal(err)
	}

	var units []twitchdl.DownloadUnit
	for _, prompt := range prompts {
		levelPropmts(&prompt, &inputPrompt)
		unit := dl.NewUnit(prompt.Input, prompt.Quality, prompt.Output, prompt.Start, prompt.End)
		units = append(units, unit)
	}

	return units
}

func processFlagInput(dl *twitchdl.Downloader, prompt Prompt) []twitchdl.DownloadUnit {
	urls := strings.Split(prompt.Input, ",")
	var units []twitchdl.DownloadUnit
	for _, url := range urls {
		prompt.Input = url
		unit := dl.NewUnit(url, prompt.Quality, prompt.Output, prompt.Start, prompt.End)
		units = append(units, unit)
	}
	return units
}

func process(dl *twitchdl.Downloader, p Prompt) []twitchdl.DownloadUnit {
	if p.Input == "" {
		log.Fatalf("Input was not provided.")
	}
	var units []twitchdl.DownloadUnit
	_, err := url.ParseRequestURI(p.Input)
	if err == nil {
		units = processFlagInput(dl, p)
	} else {
		units = processFileInput(dl, p)
	}
	return units
}

func ParseFlags(dl *twitchdl.Downloader, jsonCfg *config.Data) []twitchdl.DownloadUnit {
	var prompt Prompt

	flag.StringVar(&prompt.Input, "input", "", "Provide URL of VOD, clip or livestream to download. You can provide multiple URLs by seperating them with comma. Example: -input=https://www.twitch.tv/videos/2280187162,https://www.twitch.tv/brittt/clip/IronicArtisticOrcaWTRuck-UecXBrM6ECC-DAZR")
	flag.StringVar(&prompt.Output, "output", jsonCfg.Downloader.Output, "Path where to store the downloaded media.")
	flag.StringVar(&prompt.Quality, "quality", "", "[best 1080 720 480 360 160 worst]. Example: -quality 1080p (optional)")
	flag.DurationVar(&prompt.Start, "start", time.Duration(0), "The start of the VOD subset. It only works with VODs and it needs to be in this format: '1h30m0s' (optional)")
	flag.DurationVar(&prompt.End, "end", time.Duration(0), "The end of the VOD subset. It only works with VODs and it needs to be in this format: '1h33m0s' (optional)")
	flag.Parse()

	if prompt.Input == "" {
		if len(os.Args) > 1 {
			prompt.Input = os.Args[1]
		}
	}

	return process(dl, prompt)
}
