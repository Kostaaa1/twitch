package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Kostaaa1/twitch/internal/config"
)

type API struct {
	client *http.Client
	config config.Data
	// progressCh chan spinner.ChannelMessage
}

const (
	gqlURL      = "https://gql.twitch.tv/gql"
	gqlClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	usherURL    = "https://usher.ttvnw.net"
	helixURL    = "https://api.twitch.tv/helix"
)

func New() *API {
	return &API{
		client: http.DefaultClient,
		// progressCh: nil,
	}
}

func (tw *API) SetConfig(cfg config.Data) {
	tw.config = cfg
}

func (tw *API) do(req *http.Request) (*http.Response, error) {
	resp, err := tw.client.Do(req)
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

func (tw *API) fetchWithCode(url string) ([]byte, int, error) {
	resp, err := tw.client.Get(url)
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

func (tw *API) fetch(url string) ([]byte, error) {
	resp, err := tw.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-success HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}
	return bytes, nil
}

func (tw *API) decodeJSONResponse(resp *http.Response, p interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return err
	}
	return nil
}

func (tw *API) sendGqlLoadAndDecode(body *strings.Reader, v any) error {
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

func (tw *API) GetToken() string {
	return fmt.Sprintf("Bearer %s", tw.config.User.Creds.AccessToken)
}
