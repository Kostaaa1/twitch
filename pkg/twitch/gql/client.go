package gql

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	gqlURL      = "https://gql.twitch.tv/gql"
	gqlClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	usherURL    = "https://usher.ttvnw.net"
)

type Client struct {
	http *http.Client
}

func sendGqlLoadAndDecode[T any](
	ctx context.Context,
	c *http.Client,
	dst *T,
	gqlLoad string,
	a ...any,
) error {
	type response struct {
		Data       T `json:"data"`
		Extensions struct {
			DurationMilliseconds int    `json:"durationMilliseconds"`
			OperationName        string `json:"operationName"`
			RequestID            string `json:"requestID"`
		} `json:"extensions"`
	}

	var resp response

	var r io.Reader
	if len(a) > 0 {
		r = strings.NewReader(fmt.Sprintf(gqlLoad, a...))
	} else {
		r = strings.NewReader(gqlLoad)
	}

	h := http.Header{}
	h.Set("Client-Id", gqlClientID)
	h.Set("Content-Type", "application/json")

	if err := fetchWithDecode(ctx, c, gqlURL, http.MethodPost, r, &resp, h); err != nil {
		return err
	}

	*dst = resp.Data

	return nil
}
