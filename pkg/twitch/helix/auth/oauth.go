package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/zalando/go-keyring"
)

var (
	ErrOAuthCredentialsNotFound = errors.New("oauth: credentials not found in secure store")
	ErrOAuthCredentialsRequired = errors.New("oauth: credentials are required")
	ErrClientIDRequired         = errors.New("oauth: client ID is required")
	ErrRedirectURLRequired      = errors.New("oauth: redirect URL is required")
	ErrClientSecretRequired     = errors.New("oauth: client secret is required")
)

type OAuthCreds struct {
	ClientID     string     `json:"client_id"`
	RedirectURL  string     `json:"redirect_url"`
	ClientSecret string     `json:"client_secret,omitempty"`
	AppToken     *AppToken  `json:"app_secret,omitempty"`
	UserToken    *UserToken `json:"user_secret,omitempty"`
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
		return ErrClientIDRequired
	}
	if c.ClientSecret == "" {
		return ErrClientSecretRequired
	}
	if c.RedirectURL == "" {
		return ErrRedirectURLRequired
	}
	if c.AppToken != nil {
		if err := c.AppToken.Validate(); err != nil {
			return err
		}
	}
	if c.UserToken != nil {
		if err := c.UserToken.Validete(); err != nil {
			return err
		}
	}
	return nil
}

const (
	service     = "TwitchCLI"
	authDataKey = "twitch_oauth_credentials"
)

func (creds *OAuthCreds) Store() error {
	if err := creds.Validate(); err != nil {
		return err
	}
	jsonb, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return keyring.Set(service, authDataKey, string(jsonb))
}

func (creds *OAuthCreds) Read() error {
	v, err := keyring.Get(service, authDataKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrOAuthCredentialsNotFound
		}
		return err
	}
	return json.Unmarshal([]byte(v), &creds)
}
