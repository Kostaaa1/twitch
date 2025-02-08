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

	// msgChan := make(chan interface{})
	// ws, err := chat.CreateWSClient()
	// if err != nil {
	// 	panic(err)
	// }
	// go func() {
	// 	if err := ws.Connect(jsonCfg.User.Creds.AccessToken, jsonCfg.User.DisplayName, msgChan, []string{"ohnepixel", "tyler1"}); err != nil {
	// 		fmt.Println("Connection error: ", err)
	// 	}
	// }()
	// for msg := range msgChan {
	// 	fmt.Println("received msg: ", msg)
	// }
}
