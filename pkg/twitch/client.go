package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	http       *http.Client
	creds      *Creds
	retryCount int
}

const (
	gqlURL      = "https://gql.twitch.tv/gql"
	gqlClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	usherURL    = "https://usher.ttvnw.net"
	helixURL    = "https://api.twitch.tv/helix"
	oauthURL    = "https://id.twitch.tv/oauth2"
)

func NewClient(creds *Creds) *Client {
	return &Client{
		creds:      creds,
		retryCount: 3,
		http:       http.DefaultClient,
		// httpClient: &http.Client{
		// 	// Timeout: 15 * time.Second,
		// 	Transport: &http.Transport{
		// 		MaxIdleConns:          100,
		// 		MaxIdleConnsPerHost:   100,
		// 		IdleConnTimeout:       90 * time.Second,
		// 		TLSHandshakeTimeout:   10 * time.Second,
		// 		ExpectContinueTimeout: 1 * time.Second,
		// 	},
		// },
	}
}

func (tw *Client) HttpClient() *http.Client {
	return tw.http
}

func (tw *Client) SetHttpClient(c *http.Client) {
	tw.http = c
}

func (tw *Client) fetchWithCode(url string) ([]byte, int, error) {
	resp, err := tw.http.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("fetching failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body failed: %w", err)
	}

	return bytes, resp.StatusCode, nil
}

func (tw *Client) fetch(url string) ([]byte, error) {
	b, _, err := tw.fetchWithCode(url)
	return b, err
}

func (tw *Client) decodeJSONResponse(resp *http.Response, p interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return err
	}
	return nil
}

func (tw *Client) sendGqlLoadAndDecode(body *strings.Reader, v any) error {
	req, err := http.NewRequest(http.MethodPost, gqlURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request to get the access token: %s", err)
	}
	req.Header.Set("Client-Id", gqlClientID)

	resp, err := tw.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unsupported response status code for graphql: %v", resp.StatusCode)
	}

	if err := tw.decodeJSONResponse(resp, &v); err != nil {
		return err
	}

	return nil
}
