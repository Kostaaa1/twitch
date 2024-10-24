package main

import (
	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		panic(err)
	}
	tw := twitch.New()
	chat.Open(tw, cfg)
}
