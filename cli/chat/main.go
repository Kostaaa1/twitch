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

func authorize(creds config.Creds) error {
	if creds.ClientID == "" {
		return errors.New("Error: Client-ID is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Client-ID in config.")
	}

	if creds.RedirectURL == "" {
		return errors.New("Error: Redirect URL is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Redirect URL in config.")
	}

	if creds.RefreshToken == "" {
		codeURL := fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=channel:manage:redemptions+channel:read:redemptions+channel:read:subscriptions+moderator:read:chatters+channel:read:hype_train+bits:read", creds.ClientID, creds.RedirectURL)

		u, err := url.Parse(creds.RedirectURL)
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
				values.Add("client_id", creds.ClientID)
				values.Add("client_secret", creds.ClientSecret)
				values.Add("grant_type", "authorization_code")
				values.Add("redirect_uri", creds.RedirectURL)

				resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", values)
				if err != nil {
					log.Fatalf("failed to exchange code for refresh token: %v", err)
				}
				defer resp.Body.Close()

				var body struct {
					AccessToken  string   `json:"access_token"`
					ExpiresIn    int      `json:"expires_in"`
					RefreshToken string   `json:"refresh_token"`
					Scopes       []string `json:"scope"`
					TokenType    string   `json:"token_type"`
				}

				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					log.Fatalf("failed to decode the exchange response: %v", err)
				}

				fmt.Println(body)
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

	fmt.Println(jsonCfg.User.Creds)

	if err := authorize(jsonCfg.User.Creds); err != nil {
		log.Fatal(err)
	}
	// tw := twitch.New()
	// chat.Open(tw, jsonCfg)
}
