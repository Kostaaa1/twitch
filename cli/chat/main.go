package main

import (
	"fmt"

	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
)

// import (
// 	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
// 	"github.com/Kostaaa1/twitch/internal/config"
// 	"github.com/Kostaaa1/twitch/pkg/twitch"
// )

// func main() {
// 	cfg, err := config.Get()
// 	if err != nil {
// 		panic(err)
// 	}
// 	tw := twitch.New()
// 	chat.Open(tw, cfg)
// }

func main() {
	msgChan := make(chan interface{})

	ws, err := chat.CreateWSClient()
	if err != nil {
		panic(err)
	}

	go func() {
		if err := ws.Connect("0q5aotb6xvbdvltyz74t2ysjhbwgy3", "8lu60q33jxsrwjs3m19ktewx8y1ohs", msgChan, []string{"mizkif"}); err != nil {
			fmt.Println("Connection error: ", err)
		}
	}()

	ws.SendMessage([]byte("https://www.amazon.com/hz/wishlist/ls/1YW7EDGP4QKJB/ref=nav_wishlist_lists_1 Test12334"))

	for {
		select {
		case msg := <-msgChan:
			fmt.Println(msg)
		}
	}
}
