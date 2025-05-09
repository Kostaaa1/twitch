package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kostaaa1/twitch/internal/config"
)

type Client struct {
	httpClient *http.Client
	config     *config.Config
}

const (
	gqlURL      = "https://gql.twitch.tv/gql"
	gqlClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	usherURL    = "https://usher.ttvnw.net"
	helixURL    = "https://api.twitch.tv/helix"
	oauthURL    = "https://id.twitch.tv/oauth2"
)

func New() *Client {
	return &Client{
		httpClient: http.DefaultClient,
	}
}

func (tw *Client) Config() *config.Config {
	return tw.config
}

func (tw *Client) SetConfig(cfg *config.Config) {
	tw.config = cfg
}

func (tw *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := tw.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %s", err)
	}
	if s := resp.StatusCode; s < 200 || s >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status code %d: %s", s, string(b))
	}
	return resp, nil
}

func (tw *Client) fetchWithCode(url string) ([]byte, int, error) {
	resp, err := http.Get(url)
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

	resp, err := tw.do(req)
	if err != nil {
		return err
	}

	if err := tw.decodeJSONResponse(resp, &v); err != nil {
		return err
	}

	return nil
}

func (tw *Client) GetBearerToken() string {
	return fmt.Sprintf("Bearer %s", tw.config.Creds.AccessToken)
}

func (tw *Client) FetchAccesToken() error {
	v := url.Values{}
	v.Add("client_id", tw.config.Creds.ClientID)
	v.Add("client_secret", tw.config.Creds.ClientSecret)
	v.Add("refresh_token", tw.config.Creds.RefreshToken)
	v.Add("grant_type", "refresh_token")

	resp, err := tw.httpClient.PostForm("https://id.twitch.tv/oauth2/token", v)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(&tw.config.Creds)
}
