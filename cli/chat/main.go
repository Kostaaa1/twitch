package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

func authorize(tw *twitch.Client, conf *config.Config) error {
	if conf.Creds.ClientID == "" {
		return errors.New("error: Client-ID is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Client-ID in config")
	}

	if conf.Creds.RedirectURL == "" {
		return errors.New("error: Redirect URL is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Redirect URL in config")
	}

	if conf.Creds.RefreshToken == "" {
		codeURL := fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=channel:manage:redemptions+channel:read:redemptions+channel:read:subscriptions+moderator:read:chatters+channel:read:hype_train+bits:read+chat:read+chat:edit", conf.Creds.ClientID, conf.Creds.RedirectURL)

		u, err := url.Parse(conf.Creds.RedirectURL)
		if err != nil {
			return err
		}
		srv := &http.Server{Addr: ":" + u.Port()}

		fmt.Printf("Please visit this link to authorize: \n%s\n", codeURL)

		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code != "" {
				values := url.Values{}
				values.Add("code", code)
				values.Add("client_id", conf.Creds.ClientID)
				values.Add("client_secret", conf.Creds.ClientSecret)
				values.Add("grant_type", "authorization_code")
				values.Add("redirect_uri", conf.Creds.RedirectURL)

				resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", values)
				if err != nil {
					log.Fatalf("failed to exchange code for refresh token: %v", err)
				}
				defer resp.Body.Close()

				if err := json.NewDecoder(resp.Body).Decode(&conf.Creds); err != nil {
					log.Fatalf("failed to decode the exchange response: %v", err)
				}

				user, err := tw.User(nil, nil)
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

				// if err := config.Save(conf); err != nil {
				// 	log.Fatal(err)
				// }

				fmt.Println("Successful authorization! 🚀")
			} else {
				fmt.Println("failed to get the authorization code")
			}

			// why in separate goroutine?
			go func() {
				time.Sleep(time.Second * 1)
				if err := srv.Shutdown(context.Background()); err != nil {
					log.Printf("server shutdown error: %v", err)
				}
			}()
		})

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}

	return nil
}

func main() {
	conf, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	tw := twitch.New()
	tw.SetConfig(conf)

	if err := authorize(tw, conf); err != nil {
		log.Fatal(err)
	}
	chat.Open(tw, conf)
}
