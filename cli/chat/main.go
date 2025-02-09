package main

import (
	"log"

	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func main() {
	jsonCfg, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}
	tw := twitch.New()
	chat.Open(tw, jsonCfg)
}
