package main

import (
	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

// "creds": {
//       "accessToken": "rgtyj73qol873r67tcb7u6jade5cao",
//       "clientId": "8lu60q33jxsrwjs3m19ktewx8y1ohs",
//       "clientSecret": "bvjgex1acc8wx0c1g4qnumtxnwhdgl"
//     }

func main() {
	cfg, err := config.Get()
	if err != nil {
		panic(err)
	}
	tw := twitch.New()
	chat.Open(tw, cfg)
}
