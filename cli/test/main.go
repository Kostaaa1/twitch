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
	tw.SetConfig(conf)

	username := "piratesoftware"
	user, err := tw.User(nil, &username)
	if err != nil {
		log.Fatal(err)
	}

	// evt :=
	// evt2 :=
	// evt3 := event.StreamOnlineEvent(user.ID)
	events := []event.Event{
		event.StreamOnlineEvent(user.ID),
		// event.ChannelAdBreakBeginEvent(user.ID),
		// event.ChannelSubscribeEvent(user.ID),
		// event.ChannelUpdateEvent(user.ID),
		// event.ChannelFollowEvent(user.ID),
	}

	sub := event.NewSub(tw)
	if err := sub.DialWSS(events, time.Second*10); err != nil {
		log.Fatal(err)
	}
}
