package helix

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/Kostaaa1/twitch/internal/httputil"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix/auth"
)

func (h *Client) AppAccessToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {h.oauthCreds.ClientID},
		"client_secret": {h.oauthCreds.ClientSecret},
		"grant_type":    {"client_credentials"},
	}

	url, _ := url.Parse("https://id.twitch.tv/oauth2/token")
	url.RawQuery = values.Encode()

	header := http.Header{}
	header.Add("Content-Type", " x-www-form-urlencoded")

	return httputil.DoJSON(
		ctx,
		h.http,
		url.String(),
		http.MethodPost,
		nil,
		&h.oauthCreds.AppToken,
		header,
	)
}

func (h *Client) RefreshAccessToken(ctx context.Context) error {
	values := url.Values{
		"client_id":     {h.oauthCreds.ClientID},
		"client_secret": {h.oauthCreds.ClientSecret},
		"refresh_token": {h.oauthCreds.UserToken.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	url := "https://id.twitch.tv/oauth2/token?" + values.Encode()

	return httputil.DoJSON(
		ctx,
		h.http,
		url,
		http.MethodPost,
		nil,
		&h.oauthCreds.UserToken,
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
	err := h.RequestWithAccessToken(ctx, url, http.MethodGet, nil, &userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}

func (h *Client) RevokeAccessToken(ctx context.Context) error {
	at := h.oauthCreds.UserToken.AccessToken
	if at == "" {
		return errors.New("failed to revoke access token: not provided")
	}

	values := url.Values{"client_id": {h.oauthCreds.ClientID}, "token": {at}}
	url := "https://id.twitch.tv/oauth2/revoke?" + values.Encode()

	headers := http.Header{"Content-Type": {"x-www-form-urlencoded"}}
	return httputil.DoJSON(ctx, h.http, url, http.MethodPost, nil, nil, headers)
}

type ValidatedAccessToken struct {
	ClientID  string   `json:"client_id"`
	Login     string   `json:"login"`
	Scopes    []string `json:"scopes"`
	UserID    string   `json:"user_id"`
	ExpiresIn int      `json:"expires_in"`
}

func (h *Client) ValidateAccessToken(ctx context.Context) (*ValidatedAccessToken, error) {
	vURL := "https://id.twitch.tv/oauth2/validate"
	var resp ValidatedAccessToken
	err := h.RequestWithAccessToken(ctx, vURL, http.MethodGet, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (h *Client) UserTokenWithAuthorizationCode(ctx context.Context, code string) error {
	values := url.Values{
		"code":          {code},
		"client_id":     {h.oauthCreds.ClientID},
		"client_secret": {h.oauthCreds.ClientSecret},
		"redirect_uri":  {h.oauthCreds.RedirectURL},
		"grant_type":    {"authorization_code"},
	}
	headers := http.Header{}
	headers.Set("Content-Type", "x-www-form-urlencoded")
	return httputil.DoJSON(
		ctx,
		h.http,
		fmt.Sprintf("https://id.twitch.tv/oauth2/token?%s", values.Encode()),
		http.MethodPost,
		nil,
		&h.oauthCreds.UserToken,
		nil,
	)
}

func (h *Client) Authorize(ctx context.Context, opts auth.AuthOpts) error {
	if err := h.oauthCreds.Validate(); err != nil {
		return err
	}

	codeVals, err := h.oauthCreds.CodeExchangeURLValues(opts)
	if err != nil {
		return err
	}

	authURL, err := url.Parse("https://id.twitch.tv/oauth2/authorize")
	if err != nil {
		return err
	}
	authURL.RawQuery = codeVals.Encode()

	fmt.Printf("Please navigate to this link in browser to authorize: \n%s\n", authURL)

	redirectURL, err := url.Parse(h.oauthCreds.RedirectURL)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + redirectURL.Port(),
		Handler: mux,
	}

	pattern := "/"
	if len(redirectURL.Path) > 0 {
		pattern = redirectURL.Path
	}

	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := srv.Shutdown(context.Background()); err != nil {
				log.Printf("server shutdown error: %v", err)
			}
		}()

		queryVals := r.URL.Query()
		vstate := codeVals.Get("state")
		qstate := queryVals.Get("state")
		code := queryVals.Get("code")
		err := queryVals.Get("error")

		if vstate != qstate {
			panic(fmt.Errorf("oauth states do not match - (%s - %s) CSRF attempt\n", vstate, qstate))
		}
		if err != "" {
			errDesc := queryVals.Get("error_description")
			panic(errors.Join(errors.New(errDesc), errors.New(err)))
		}
		if code == "" {
			panic("code is empty")
		}
		if err := h.UserTokenWithAuthorizationCode(ctx, code); err != nil {
			panic(err)
		}

		fmt.Println("Successful authorization! 🚀")
	})

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
