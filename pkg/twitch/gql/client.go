package gql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Kostaaa1/twitch/internal/httputil"
)

const (
	gqlURL   = "https://gql.twitch.tv/gql"
	clientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
)

type Client struct {
	http *http.Client
}

type GQLError struct {
	Message string `json:"message,omitempty"`
}

type GQLResponse[T any] struct {
	Data       T          `json:"data"`
	Errors     []GQLError `json:"errors,omitempty"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{http: httpClient}
}

type PersistedQuery struct {
	Version    int    `json:"version"`
	Sha256Hash string `json:"sha256Hash"`
}

type Extensions struct {
	PersistedQuery PersistedQuery `json:"persistedQuery"`
}

type Payload struct {
	OperationName string         `json:"operationName,omitempty"`
	Query         string         `json:"query,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
	Extensions    *Extensions    `json:"extensions,omitempty"`
}

func commonExtensions(sha256hash string) *Extensions {
	return &Extensions{
		PersistedQuery: PersistedQuery{
			Version:    1,
			Sha256Hash: sha256hash,
		},
	}
}

func sendGqlLoadAndDecode[T any](
	ctx context.Context,
	httpClient *http.Client,
	dst *T,
	payload Payload,
) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(b)

	h := http.Header{}
	h.Set("Client-Id", clientID)
	h.Set("Content-Type", "application/json")

	var resp GQLResponse[T]
	if err := httputil.DoJSON(
		ctx,
		httpClient,
		gqlURL,
		http.MethodPost,
		reader,
		&resp,
		h,
	); err != nil {
		return err
	}

	if len(resp.Errors) > 0 {
		var err error
		for _, gqlerr := range resp.Errors {
			err = errors.Join(err, errors.New(gqlerr.Message))
		}
		return err
	}
	*dst = resp.Data

	return nil
}
