package helix

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

func (h *Client) AppToken(ctx context.Context) error {
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
		&h.OAuthCreds.UserToken,
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
	return h.UserTokenWithRefreshToken(ctx)
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

func (h *Client) authWithCodeURL() string {
	values := url.Values{
		"client_id":    {h.OAuthCreds.ClientID},
		"redirect_url": {h.OAuthCreds.RedirectURL},
		"scope":        {strings.Join(defaultScope, " ")},
		// "state":        {},
	}
	return fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&%s", values.Encode())
}

func (h *Client) authWithTokenURL() string {
	values := url.Values{
		"client_id":    {h.OAuthCreds.ClientID},
		"redirect_url": {h.OAuthCreds.RedirectURL},
		"scope":        {strings.Join(defaultScope, " ")},
		// "state":        {},
	}
	return fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=token&%s", values.Encode())
}

type authResponseType string

const (
	TokenResponseType authResponseType = "token"
	CodeResponseType  authResponseType = "code"
)

type AuthOpts struct {
	ForceVerify  bool
	Scopes       []Scope
	ResponseType authResponseType
}

func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *Client) URLValues(opts AuthOpts) (url.Values, error) {
	scopes := defaultScope

	if len(opts.Scopes) > 0 {
		scopes = make([]string, len(opts.Scopes))
		for i, scope := range opts.Scopes {
			scopes[i] = string(scope)
		}
	}

	state, err := generateState()
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"response_type": {string(opts.ResponseType)},
		"client_id":     {h.OAuthCreds.ClientID},
		"redirect_uri":  {h.OAuthCreds.RedirectURL},
		"scope":         {strings.Join(scopes, " ")},
		"state":         {state},
	}

	if opts.ForceVerify {
		values.Add("force_verify", "true")
	}

	return values, nil
}

func (h *Client) Authorize(ctx context.Context, opts AuthOpts) error {
	if err := h.ensureValidCreds(ctx); err != nil {
		return err
	}

	values, err := h.URLValues(opts)
	if err != nil {
		return err
	}

	authURL, err := url.Parse("https://id.twitch.tv/oauth2/authorize")
	if err != nil {
		return err
	}
	authURL.RawQuery = values.Encode()

	fmt.Println("This is the URL", authURL)

	redirectURL, err := url.Parse(h.OAuthCreds.RedirectURL)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	srv := &http.Server{Addr: ":" + redirectURL.Port(), Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Redirected:", r.URL.Path)
		fmt.Println("value state:", values.Get("state"))
		fmt.Println("query state:", r.URL.Query().Get("state"))

		// if values.Get("state") != r.URL.Query().Get("state") {
		// 	// block
		// 	fmt.Println("CSRF attempt - states do not match")
		// 	return
		// }

		// switch opts.ResponseType {
		// case TokenResponseType:
		// 	code := r.URL.Query().Get("code")
		// 	fmt.Println("code: ", code)
		// case CodeResponseType:
		// 	token := r.URL.Query().Get("token")
		// 	fmt.Println("token: ", token)
		// }

		// code := r.URL.Query().Get(string(opts.Type))
		// if code != "" {
		// 	if err := h.UserTokenWithAuthorizationCode(ctx, code); err != nil {
		// 		log.Fatal(err)
		// 	}
		// 	fmt.Println("Successful authorization! 🚀")
		// } else {
		// 	fmt.Println("failed to get the authorization code")
		// }

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

	return nil
}

// func (h *Client) Authorize(ctx context.Context) error {
// 	if err := h.ensureValidCreds(ctx); err != nil {
// 		return err
// 	}

// 	if h.OAuthCreds.UserToken.RefreshToken == "" {
// 		values := url.Values{
// 			"client_id":    {h.OAuthCreds.ClientID},
// 			"redirect_uri": {h.OAuthCreds.RedirectURL},
// 			"scope":        {strings.Join(defaultScope, " ")},
// 		}
// 		codeURL := fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?response_type=code&%s", values.Encode())

// 		// what := "https://id.twitch.tv/oauth2/authorize?response_type=token&client_id=hof5gwx0su6owfnys0yan9c87zr6t&redirect_uri=http://localhost:3000&scope=channel%3Amanage%3Apolls+channel%3Aread%3Apolls&state=c3ab8aa609ea11e793ae92361f002671"

// 		redirectURL, err := url.Parse(h.OAuthCreds.RedirectURL)
// 		if err != nil {
// 			return err
// 		}

// 		mux := http.NewServeMux()
// 		srv := &http.Server{Addr: ":" + redirectURL.Port(), Handler: mux}

// 		fmt.Printf("Please visit this link to authorize: \n%s\n", codeURL)

// 		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
// 			code := r.URL.Query().Get("code")

// 			if code != "" {
// 				if err := h.UserTokenWithAuthorizationCode(ctx, code); err != nil {
// 					log.Fatal(err)
// 				}
// 				fmt.Println("Successful authorization! 🚀")
// 			} else {
// 				fmt.Println("failed to get the authorization code")
// 			}

// 			go func() {
// 				time.Sleep(time.Second * 1)
// 				if err := srv.Shutdown(context.Background()); err != nil {
// 					log.Printf("server shutdown error: %v", err)
// 				}
// 			}()
// 		})

// 		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
// 			return err
// 		}
// 	}

// 	return nil
// }
