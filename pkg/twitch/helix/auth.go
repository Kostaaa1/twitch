package helix

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/httputil"
)

type AppToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type UserToken struct {
	RefreshToken string   `json:"refresh_token"`
	AccessToken  string   `json:"access_token"`
	Scope        []string `json:"scope"`
	ExpiresIn    int      `json:"expires_in"`
	TokenType    string   `json:"token_type"`
}

type OAuthCreds struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	RedirectURL  string    `json:"redirect_url"`
	AppToken     AppToken  `json:"app_token"`
	UserToken    UserToken `json:"user_token"`
}

var (
	oauthMissingAccessTokenErr = errors.New("oauth creds error: refresh token is present but access token is not - refetch it")
)

func (h *Client) FetchAppToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {h.OAuthCreds.ClientID},
		"client_secret": {h.OAuthCreds.ClientSecret},
		"grant_type":    {"client_credentials"},
	}

	if err := httputil.FetchWithDecode(
		ctx,
		h.http,
		"https://id.twitch.tv/oauth2/token&"+values.Encode(),
		http.MethodPost,
		nil,
		&h.OAuthCreds,
		nil,
	); err != nil {
		return err
	}

	return nil
}

func (h *Client) UserTokenWithRefreshToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {h.OAuthCreds.ClientID},
		"client_secret": {h.OAuthCreds.ClientSecret},
		"refresh_token": {h.OAuthCreds.UserToken.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	if err := httputil.FetchWithDecode(
		ctx,
		h.http,
		fmt.Sprintf("https://id.twitch.tv/oauth2/token?%s", values.Encode()),
		http.MethodPost,
		nil,
		&h.OAuthCreds,
		nil,
	); err != nil {
		return err
	}

	return nil
}

func (h *Client) UserTokenWithAuthorizationCode(ctx context.Context, code string) error {
	values := url.Values{
		"code":          {code},
		"client_id":     {h.OAuthCreds.ClientID},
		"client_secret": {h.OAuthCreds.ClientSecret},
		"redirect_uri":  {h.OAuthCreds.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	if err := httputil.FetchWithDecode(
		ctx,
		h.http,
		fmt.Sprintf("https://id.twitch.tv/oauth2/token?%s", values.Encode()),
		http.MethodPost,
		nil,
		&h.OAuthCreds.UserToken,
		nil,
	); err != nil {
		return err
	}

	return nil
}

func (h *Client) ensureValidCreds(ctx context.Context) error {
	// if err := h.userToken.Validate(); err != nil {
	// 	if !errors.Is(err, oauthMissingAccessTokenErr) {
	// 		return err
	// 	}
	// }
	// return h.UserTokenWithRefreshToken(ctx)
	return nil
}

var defaultScope = []string{
	"channel:manage:redemptions",
	"channel:read:hype_train",
	"channel:read:redemptions",
	"channel:read:subscriptions",
	"chat:edit",
	"chat:read",
	"moderator:read:chatters",
	"user:manage:blocked_users",
	"user:read:blocked_users",
	"user:read:follows",
	"user:read:subscriptions",
	"whispers:edit",
	"whispers:read",
}

func (h *Client) authorizeURLWithCode() string {
	values := url.Values{
		"client_id":    {h.OAuthCreds.ClientID},
		"redirect_url": {h.OAuthCreds.RedirectURL},
		"scope":        {strings.Join(defaultScope, " ")},
		// "state":        {},
	}
	return fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&%s", values.Encode())
}

func (h *Client) authorizeURLWithToken() string {
	values := url.Values{
		"client_id":    {h.OAuthCreds.ClientID},
		"redirect_url": {h.OAuthCreds.RedirectURL},
		"scope":        {strings.Join(defaultScope, " ")},
		// "state":        {},
	}
	return fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=token&%s", values.Encode())
}

func (h *Client) authorizeWithCode(ctx context.Context) error {
	return nil
}

func (h *Client) Authorize(ctx context.Context) error {
	if err := h.ensureValidCreds(ctx); err != nil {
		return err
	}

	if h.OAuthCreds.UserToken.RefreshToken == "" {
		values := url.Values{
			"client_id":    {h.OAuthCreds.ClientID},
			"redirect_uri": {h.OAuthCreds.RedirectURL},
			"scope":        {strings.Join(defaultScope, " ")},
		}

		codeURL := fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&%s", values.Encode())

		redirectURL, err := url.Parse(h.OAuthCreds.RedirectURL)
		if err != nil {
			return err
		}

		srv := &http.Server{Addr: ":" + redirectURL.Port()}

		fmt.Printf("Please visit this link to authorize: \n%s\n", codeURL)

		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")

			if code != "" {
				if err := h.UserTokenWithAuthorizationCode(ctx, code); err != nil {
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
			return err
		}
	}

	return nil
}
