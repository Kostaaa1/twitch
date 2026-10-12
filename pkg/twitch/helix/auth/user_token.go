package auth

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrRefreshTokenRequired = errors.New("oauth: access token is required")
	ErrUserTokenExpired     = errors.New("oauth: user token expired")
)

type UserToken struct {
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token"`
	Scope        []string  `json:"scope"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (t *UserToken) UnmarshalJSON(b []byte) error {
	type Alias UserToken
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

func (t *UserToken) Expired() bool {
	return t.ExpiresAt.IsZero() || time.Since(t.ExpiresAt) > 0
}

func (t *UserToken) Validete() error {
	if t.AccessToken == "" {
		return ErrAccessTokenRequired
	}
	if t.RefreshToken == "" {
		return ErrRefreshTokenRequired
	}
	if t.Expired() {
		return ErrTokenExpired
	}
	return nil
}
