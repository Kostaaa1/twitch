package auth

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrAccessTokenRequired = errors.New("oauth: access token is required")
	ErrTokenExpired        = errors.New("oauth: app token expired")
)

type AppToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (t *AppToken) UnmarshalJSON(b []byte) error {
	type Alias AppToken
	aux := &struct {
		*Alias
		ExpiresAt string `json:"expires_at"`
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	if aux.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, aux.ExpiresAt)
		if err != nil {
			return err
		}
		t.ExpiresAt = parsed
	} else if t.ExpiresIn > 0 {
		t.ExpiresAt = time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
	}
	return nil
}

func (t *AppToken) expired() bool {
	return t.ExpiresAt.IsZero() || time.Since(t.ExpiresAt) > 0
}

func (t *AppToken) Validate() error {
	if t.AccessToken == "" {
		return ErrAccessTokenRequired
	}
	if t.expired() {
		return ErrTokenExpired
	}
	return nil
}
