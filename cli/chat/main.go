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

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
)

// func openBrowser(uri string) {
// 	var cmd *exec.Cmd
// 	switch runtime.GOOS {
// 	case "linux":
// 		cmd = exec.Command("xdg-open", uri)
// 	case "windows":
// 		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", uri)
// 	case "darwin":
// 		cmd = exec.Command("open", uri)
// 	}
// 	if cmd != nil {
// 		pipe, pipeErr := cmd.StderrPipe()
// 		if pipeErr != nil {
// 			log.Fatalf("failed to get the stderr pipe: %v", pipeErr)
// 		}
// 		if err := cmd.Start(); err != nil {
// 			log.Fatalf("command start fail: %v", err)
// 		}
// 		buf := make([]byte, 1024)
// 		_, err := pipe.Read(buf)
// 		if err != nil {
// 			log.Fatalf("failed to read the stderr pipe to buffer: %v", err)
// 		}
// 		fmt.Println("OUTPUT: ", string(buf))
// 		if len(buf) > 0 {
// 			fmt.Printf("Could not open the browser. Please visit this link: \n%s\n", uri)
// 		}
// 	}
// }

func authorize(tw *twitch.Client, conf *config.Config) error {
	if conf.User.Creds.ClientID == "" {
		return errors.New("Error: Client-ID is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Client-ID in config.")
	}

	if conf.User.Creds.RedirectURL == "" {
		return errors.New("Error: Redirect URL is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Redirect URL in config.")
	}

	if conf.User.Creds.RefreshToken == "" {
		codeURL := fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=channel:manage:redemptions+channel:read:redemptions+channel:read:subscriptions+moderator:read:chatters+channel:read:hype_train+bits:read", conf.User.Creds.ClientID, conf.User.Creds.RedirectURL)

		u, err := url.Parse(conf.User.Creds.RedirectURL)
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
				values.Add("client_id", conf.User.Creds.ClientID)
				values.Add("client_secret", conf.User.Creds.ClientSecret)
				values.Add("grant_type", "authorization_code")
				values.Add("redirect_uri", conf.User.Creds.RedirectURL)

				resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", values)
				if err != nil {
					log.Fatalf("failed to exchange code for refresh token: %v", err)
				}
				defer resp.Body.Close()

				if err := json.NewDecoder(resp.Body).Decode(&conf.User.Creds); err != nil {
					log.Fatalf("failed to decode the exchange response: %v", err)
				}

				path, err := config.GetConfigPath()
				if err != nil {
					log.Fatalf("failed to get the config path: %v\n", err)
				}

				user, err := tw.GetUserInfo(nil, nil)
				if err != nil {
					log.Fatal("failed to get the user info: %v\n", err)
				}

				conf.User

				if err := config.Save(path, *conf); err != nil {
					log.Fatal(err)
				}

				fmt.Println("Successful authorization! 🚀")
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
	jsonCfg, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}
	tw := twitch.New()
	tw.SetConfig(jsonCfg)
	if err := authorize(tw, &jsonCfg); err != nil {
		log.Fatal(err)
	}
	// chat.Open(tw, jsonCfg)
}
