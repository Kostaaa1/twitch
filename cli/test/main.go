package main

import (
	"log"

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
	tw.SetConfig(conf)

	sub := event.NewSub(tw)

	evt, err := sub.StreamOnlineEvent("zackrawrr")
	if err != nil {
		log.Fatal(err)
	}
	evt2, err := sub.StreamOnlineEvent("nmplol")
	if err != nil {
		log.Fatal(err)
	}
	evt3, err := sub.StreamOnlineEvent("mizkif")
	if err != nil {
		log.Fatal(err)
	}
	events := []event.Event{evt, evt2, evt3}

	if err := sub.DialWSS(events); err != nil {
		log.Fatal(err)
	}
}
