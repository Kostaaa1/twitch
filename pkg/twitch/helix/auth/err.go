package auth

import "errors"

var (
	ErrMissingOAuthCredentials = errors.New("must provide OAuth credentials")
	ErrMissingClientID         = errors.New("client ID is missing")
	ErrMissingClientSecret     = errors.New("client secret is missing")
	ErrMissingRedirectURL      = errors.New("redirect url is missing")
)
