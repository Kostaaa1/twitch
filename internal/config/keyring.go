package config

import (
	"encoding/json"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix/auth"
	"github.com/zalando/go-keyring"
)

const (
	service     = "TwitchCLI"
	authDataKey = "twitch_oauth_credentials"
)

type SecureValues struct {
	AccessToken  string
	RefreshToken string
	ClientSecret string
}

func StoreOAuthCreds(creds auth.OAuthCreds) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	jsonb, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return keyring.Set(service, authDataKey, string(jsonb))
}

func GetOAuthCreds() (*auth.OAuthCreds, error) {
	v, err := keyring.Get(service, authDataKey)
	if err != nil {
		return nil, err
	}
	var data auth.OAuthCreds
	if err := json.Unmarshal([]byte(v), &data); err != nil {
		return nil, err
	}
	return &data, err
}
