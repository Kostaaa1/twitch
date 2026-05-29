package httputil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func Do(
	ctx context.Context,
	c *http.Client,
	url string,
	method string,
	body io.Reader,
	h http.Header,
) (*http.Response, error) {
	if url == "" {
		return nil, errors.New("failed to fetch: missing url")
	}
	if method == "" {
		return nil, errors.New("failed to fetch: missing method")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create the request: url=%s err=%v", url, err)
	}
	req.Header = h.Clone()

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func FetchWithDecode(
	ctx context.Context,
	httpClient *http.Client,
	url string,
	method string,
	body io.Reader,
	dst any,
	h http.Header,
) error {
	if dst == nil {
		return errors.New("dst cannot be nil")
	}
	if url == "" {
		return errors.New("failed to fetch: missing url")
	}
	if method == "" {
		return errors.New("failed to fetch: missing method")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create the request: url=%s err=%v", url, err)
	}
	req.Header = h.Clone()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read the error response: %v", err)
		}
		return fmt.Errorf("invalid status %d: %s", resp.StatusCode, string(b))
	}

	if resp.Body != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return err
		}
	}

	return nil
}
