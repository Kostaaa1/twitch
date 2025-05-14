package main

import (
	"log"
	"time"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/event"
)

func main() {
	conf, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	tw := twitch.New()
	tw.SetCreds(&conf.Creds)

	user, _ := tw.UserByChannelName("piratesoftware")
	user2, _ := tw.UserByChannelName("jasontheween")
	user3, _ := tw.UserByChannelName("stableronaldo")
	user4, _ := tw.UserByChannelName("lacy")
	user5, _ := tw.UserByChannelName("kaicenat")
	user6, _ := tw.UserByChannelName("extraemily")
	user7, _ := tw.UserByChannelName("emiru")
	user8, _ := tw.UserByChannelName("cinna")

	events := []event.Event{
		event.StreamOnlineEvent(user.ID),
		event.StreamOnlineEvent(user2.ID),
		event.StreamOnlineEvent(user3.ID),
		event.StreamOnlineEvent(user4.ID),
		event.StreamOnlineEvent(user5.ID),
		event.StreamOnlineEvent(user6.ID),
		event.StreamOnlineEvent(user7.ID),
		event.StreamOnlineEvent(user8.ID),
	}

	sub := event.NewSub(tw)
	if err := sub.DialWSS(events, time.Second*10); err != nil {
		log.Fatal(err)
	}
}
