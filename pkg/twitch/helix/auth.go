package helix

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	TokenType   string    `json:"token_type"`
	ExpiryDate  time.Time `json:"expiry_date"`
}

func (at *AppToken) UnmarshalJSON(data []byte) error {
	type Alias AppToken

	aux := &struct{ *Alias }{Alias: (*Alias)(at)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if at.ExpiresIn > 0 {
		at.ExpiryDate = time.Now().Add(time.Duration(at.ExpiresIn))
	}

	return nil
}

func (ut *AppToken) Expired() bool { return time.Since(ut.ExpiryDate) > 0 }

type UserToken struct {
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token"`
	Scope        []string  `json:"scope"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	ExpiryDate   time.Time `json:"expiry_date"`
}

func (t *UserToken) UnmarshalJSON(data []byte) error {
	type Alias UserToken

	aux := &struct{ *Alias }{Alias: (*Alias)(t)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if t.ExpiresIn > 0 {
		t.ExpiryDate = time.Now().Add(time.Duration(t.ExpiresIn))
	}

	return nil
}

func (ut *UserToken) Expired() bool { return time.Since(ut.ExpiryDate) > 0 }

type OAuthCreds struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	RedirectURL  string    `json:"redirect_url"`
	AppToken     AppToken  `json:"app_token"`
	UserToken    UserToken `json:"user_token"`
}

type AuthOpts struct {
	ForceVerify bool
	Scopes      []Scope
}

func (creds *OAuthCreds) generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (creds *OAuthCreds) codeExchangeURLValues(opts AuthOpts) (url.Values, error) {
	var scopes []string

	if len(opts.Scopes) > 0 {
		scopes = make([]string, len(opts.Scopes))
		for i, scope := range opts.Scopes {
			scopes[i] = string(scope)
		}
	}

	state, err := creds.generateState()
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"response_type": {"code"},
		"client_id":     {creds.ClientID},
		"redirect_uri":  {creds.RedirectURL},
		"scope":         {strings.Join(scopes, " ")},
		"state":         {state},
	}

	if opts.ForceVerify {
		values.Add("force_verify", "true")
	}

	return values, nil
}

func (c *OAuthCreds) Validate() error {
	if c.ClientID == "" {
		return ErrMissingClientID
	}
	if c.ClientSecret == "" {
		return ErrMissingClientSecret
	}
	if c.RedirectURL == "" {
		return ErrMissingRedirectURL
	}
	return nil
}

var (
	ErrMissingClientID     = errors.New("client ID is missing")
	ErrMissingClientSecret = errors.New("client secret is missing")
	ErrMissingRedirectURL  = errors.New("redirect url is missing")
)

func (h *Client) AppAccessToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {h.OAuthCreds.ClientID},
		"client_secret": {h.OAuthCreds.ClientSecret},
		"grant_type":    {"client_credentials"},
	}

	url, _ := url.Parse("https://id.twitch.tv/oauth2/token")
	url.RawQuery = values.Encode()

	header := http.Header{}
	header.Add("Content-Type", " x-www-form-urlencoded")

	return httputil.FetchWithDecode(
		ctx,
		h.http,
		url.String(),
		http.MethodPost,
		nil,
		&h.OAuthCreds.AppToken,
		header,
	)
}

func (h *Client) RefreshAccessToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {h.OAuthCreds.ClientID},
		"client_secret": {h.OAuthCreds.ClientSecret},
		"refresh_token": {h.OAuthCreds.UserToken.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	url := "https://id.twitch.tv/oauth2/token?" + values.Encode()

	return httputil.FetchWithDecode(
		ctx,
		h.http,
		url,
		http.MethodPost,
		nil,
		&h.OAuthCreds.UserToken,
		nil,
	)
}

type UserInfo struct {
	Aud           string    `json:"aud"`
	Exp           int       `json:"exp"`
	Iat           int       `json:"iat"`
	Iss           string    `json:"iss"`
	Sub           string    `json:"sub"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Picture       string    `json:"picture"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (h *Client) Claims(ctx context.Context) (*UserInfo, error) {
	url := "https://id.twitch.tv/oauth2/userinfo"
	var userInfo UserInfo
	if err := h.Request(ctx, url, http.MethodGet, nil, &userInfo); err != nil {
		return nil, err
	}
	return &userInfo, nil
}

func (h *Client) RevokeAccessToken(ctx context.Context) error {
	at := h.OAuthCreds.UserToken.AccessToken
	fmt.Println("AT: ", at)
	if at == "" {
		return errors.New("failed to revoke access token: not provided")
	}

	values := url.Values{"client_id": {h.OAuthCreds.ClientID}, "token": {at}}
	url := "https://id.twitch.tv/oauth2/revoke?" + values.Encode()

	headers := http.Header{"Content-Type": {"x-www-form-urlencoded"}}
	return httputil.FetchWithDecode(ctx, h.http, url, http.MethodPost, nil, nil, headers)
}

type ValidatedAccessToken struct {
	ClientID  string   `json:"client_id"`
	Login     string   `json:"login"`
	Scopes    []string `json:"scopes"`
	UserID    string   `json:"user_id"`
	ExpiresIn int      `json:"expires_in"`
}

func (h *Client) ValidateAccessToken(ctx context.Context) (*ValidatedAccessToken, error) {
	var resp ValidatedAccessToken
	if err := h.Request(ctx, "https://id.twitch.tv/oauth2/validate", http.MethodGet, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (h *Client) UserTokenWithAuthorizationCode(ctx context.Context, code string) error {
	values := url.Values{
		"code":          {code},
		"client_id":     {h.OAuthCreds.ClientID},
		"client_secret": {h.OAuthCreds.ClientSecret},
		"redirect_uri":  {h.OAuthCreds.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	headers := http.Header{}
	headers.Set("Content-Type", "x-www-form-urlencoded")

	return httputil.FetchWithDecode(
		ctx,
		h.http,
		fmt.Sprintf("https://id.twitch.tv/oauth2/token?%s", values.Encode()),
		http.MethodPost,
		nil,
		&h.OAuthCreds.UserToken,
		nil,
	)
}

func (h *Client) Authorize(ctx context.Context, opts AuthOpts) error {
	if err := h.OAuthCreds.Validate(); err != nil {
		return err
	}

	v, err := h.OAuthCreds.codeExchangeURLValues(opts)
	if err != nil {
		return err
	}

	authURL, err := url.Parse("https://id.twitch.tv/oauth2/authorize")
	if err != nil {
		return err
	}
	authURL.RawQuery = v.Encode()

	fmt.Printf("Please navigate to this link in browser to authorize: \n%s\n", authURL)

	redirectURL, err := url.Parse(h.OAuthCreds.RedirectURL)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + redirectURL.Port(),
		Handler: mux,
	}

	mux.HandleFunc(redirectURL.Path, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := srv.Shutdown(context.Background()); err != nil {
				log.Printf("server shutdown error: %v", err)
			}
		}()

		q := r.URL.Query()
		vstate, qstate, code, err := v.Get("state"), q.Get("state"), q.Get("code"), q.Get("error")

		if vstate != qstate {
			log.Fatalf("states do not match - (%s - %s) CSRF attempt\n", vstate, qstate)
		}
		if err != "" {
			errDesc := q.Get("error_description")
			log.Fatal(errors.Join(errors.New(errDesc), errors.New(err)))
		}
		if code == "" {
			log.Fatal("code is empty")
		}
		if err := h.UserTokenWithAuthorizationCode(ctx, code); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Successful authorization! 🚀")
	})

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
