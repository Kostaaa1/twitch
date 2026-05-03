package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type OAuthCreds struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	RefreshToken string   `json:"refresh_token"`
	AccessToken  string   `json:"access_token"`
	ExpiresIn    int      `json:"expires_in"`
	TokenType    string   `json:"token_type"`
	Scope        []string `json:"scope"`
}

func (tw *Client) GetBearerToken() string {
	return fmt.Sprintf("Bearer %s", tw.oauthCreds.AccessToken)
}

func (tw *Client) FetchAccesToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {tw.oauthCreds.ClientID},
		"client_secret": {tw.oauthCreds.ClientSecret},
		"refresh_token": {tw.oauthCreds.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	accessTokenEndpoint := "https://id.twitch.tv/oauth2/token?" + values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, accessTokenEndpoint, nil)
	if err != nil {
		return err
	}

	resp, err := tw.http.Do(req)
	if err != nil {
		return err
	}

	// resp, err := tw.http.PostForm("https://id.twitch.tv/oauth2/token", values)
	// if err != nil {
	// 	return err
	// }
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: status %d\nresponse: %s", resp.StatusCode, body)
	}

	err = json.NewDecoder(resp.Body).Decode(&tw.oauthCreds)
	return err
}

func (tw *Client) Authorize(ctx context.Context) error {
	if tw.oauthCreds.ClientID == "" {
		return errors.New("error: Client-ID is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Client-ID in config")
	}
	if tw.oauthCreds.RedirectURL == "" {
		return errors.New("error: Redirect URL is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Redirect URL in config")
	}
	if tw.oauthCreds.RefreshToken != "" && tw.oauthCreds.AccessToken == "" {
		if err := tw.FetchAccesToken(ctx); err != nil {
			return err
		}
	}

	if tw.oauthCreds.RefreshToken == "" {
		codeURL := fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=channel:manage:redemptions+user:manage:blocked_users+user:read:blocked_users+user:read:follows+user:read:subscriptions+whispers:edit+whispers:read+channel:read:redemptions+channel:read:subscriptions+moderator:read:chatters+channel:read:hype_train+bits:read+chat:read+chat:edit", tw.oauthCreds.ClientID, tw.oauthCreds.RedirectURL)

		redirectURL, err := url.Parse(tw.oauthCreds.RedirectURL)
		if err != nil {
			return err
		}

		srv := &http.Server{Addr: ":" + redirectURL.Port()}
		fmt.Printf("Please visit this link to authorize: \n%s\n", codeURL)

		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code != "" {
				values := url.Values{
					"code":          {code},
					"client_id":     {tw.oauthCreds.ClientID},
					"client_secret": {tw.oauthCreds.ClientSecret},
					"grant_type":    {"authorization_code"},
					"redirect_uri":  {tw.oauthCreds.RedirectURL},
				}

				// resp, err := tw.http.PostForm("https://id.twitch.tv/oauth2/token", values)
				// if err != nil {
				// 	log.Fatalf("failed to exchange code for refresh token: %v", err)
				// }
				// defer resp.Body.Close()

				tokenEndpoint := "https://id.twitch.tv/oauth2/token?" + values.Encode()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenEndpoint, nil)
				if err != nil {
					log.Fatalf("failed to exchange code for refresh token: %v", err)
				}

				resp, err := tw.http.Do(req)
				if err != nil {
					log.Fatalf("failed to exchange code for refresh token: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					body, _ := io.ReadAll(resp.Body)
					log.Fatalf("token exchange failed: status %d\nresponse: %s", resp.StatusCode, body)
				}

				if err := json.NewDecoder(resp.Body).Decode(&tw.oauthCreds); err != nil {
					log.Fatalf("failed to decode the exchange response: %v", err)
				}

				fmt.Println("Successful authorization! 🚀")
			} else {
				fmt.Println("failed to get the authorization code")
			}

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
