package main

import (
	"fmt"

	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
)

func main() {
	msgChan := make(chan interface{})

	ws, err := chat.CreateWSClient()
	if err != nil {
		panic(err)
	}

	go func() {
		if err := ws.Connect("rgtyj73qol873r67tcb7u6jade5cao", "slorpglorpski	", msgChan, []string{"mizkif", "asmongold"}); err != nil {
			fmt.Println("Connection error: ", err)
		}
	}()

	for {
		select {
		case msg := <-msgChan:
			fmt.Println(msg)
		}
	}
}
