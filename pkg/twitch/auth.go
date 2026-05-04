package twitch

import (
	"context"
	"errors"
	"fmt"
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

var (
	oauthMissingAccessTokenErr = errors.New("oauth creds error: refresh token is present but access token is not - refetch it")
)

func (creds *OAuthCreds) Validate() error {
	if creds == nil {
		return errors.New("oauth creds is nil")
	}
	if creds.ClientID == "" {
		return errors.New("oauth creds error: Client-ID is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Client-ID in config")
	}
	if creds.RedirectURL == "" {
		return errors.New("oauth creds error: Redirect URL is missing from the config file. Please create an application via dev.twitch.tv/console and provide the Redirect URL in config")
	}
	if creds.RefreshToken != "" && creds.AccessToken == "" {
		return oauthMissingAccessTokenErr
	}

	// validate access token?

	return nil
}

func (tw *Client) AccesToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {tw.oauthCreds.ClientID},
		"client_secret": {tw.oauthCreds.ClientSecret},
		"refresh_token": {tw.oauthCreds.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	accessTokenURL := "https://id.twitch.tv/oauth2/token?" + values.Encode()

	if err := tw.fetchWithDecode(
		ctx,
		accessTokenURL,
		http.MethodPost,
		nil,
		&tw.oauthCreds,
		nil,
	); err != nil {
		return err
	}

	return nil
}

func (tw *Client) ensureValidCreds(ctx context.Context) error {
	if err := tw.oauthCreds.Validate(); err != nil {
		if !errors.Is(err, oauthMissingAccessTokenErr) {
			return err
		}
		return tw.AccesToken(ctx)
	}
	return nil
}

func (tw *Client) Authorize(ctx context.Context) error {
	if err := tw.ensureValidCreds(ctx); err != nil {
		return err
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

				tokenURL := "https://id.twitch.tv/oauth2/token?" + values.Encode()

				if err := tw.fetchWithDecode(ctx, tokenURL, http.MethodGet, nil, &tw.oauthCreds, nil); err != nil {
					log.Fatal(err)
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
