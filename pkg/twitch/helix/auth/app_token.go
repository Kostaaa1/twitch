package auth

import (
	"encoding/json"
	"time"
)

type AppToken struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	TokenType   string    `json:"token_type"`
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

func (t *AppToken) Expired() bool {
	return t.ExpiresAt.IsZero() || time.Since(t.ExpiresAt) > 0
}
