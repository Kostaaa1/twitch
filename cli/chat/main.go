package main

import (
	"log"

	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func main() {
	conf, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	tw := twitch.NewClient(nil, &conf.Creds)

	if err := tw.Authorize(); err != nil {
		log.Fatal(err)
	}

	user, err := tw.UserByChannelName("")
	if err != nil {
		log.Fatalf("failed to get the user info: %v\n", err)
	}
	conf.User = config.User{
		BroadcasterType: user.BroadcasterType,
		CreatedAt:       user.CreatedAt,
		Description:     user.Description,
		DisplayName:     user.DisplayName,
		ID:              user.ID,
		Login:           user.Login,
		OfflineImageURL: user.OfflineImageURL,
		ProfileImageURL: user.ProfileImageURL,
		Type:            user.Type,
	}
	conf.Save()

	chat.Open(tw, conf)
}
