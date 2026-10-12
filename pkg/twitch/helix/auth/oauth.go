package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"strings"
)

type OAuthCreds struct {
	ClientID     string     `json:"client_id"`
	RedirectURL  string     `json:"redirect_url"`
	ClientSecret string     `json:"client_secret"`
	AppToken     *AppToken  `json:"app_secret"`
	UserToken    *UserToken `json:"user_secret"`
}

func (creds *OAuthCreds) generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type AuthOpts struct {
	ForceVerify bool
	Scopes      []Scope
}

func (creds *OAuthCreds) CodeExchangeURLValues(opts AuthOpts) (url.Values, error) {
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
