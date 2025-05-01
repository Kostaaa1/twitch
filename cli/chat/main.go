package main

import (
	"log"

	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

// getting code to
// https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=YOURAPPCLIENTID&redirect_uri=http://localhost&scope=channel:manage:redemptions+channel:read:redemptions+channel:read:subscriptions+moderator:read:chatters+channel:read:hype_train+bits:read

func main() {
	jsonCfg, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}
	tw := twitch.New()
	chat.Open(tw, jsonCfg)
}
