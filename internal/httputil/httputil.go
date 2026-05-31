package httputil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func CodeSuccess(code int) bool { return code < http.StatusOK || code >= http.StatusMultipleChoices }

func Fetch(
	ctx context.Context,
	c *http.Client,
	url string,
	method string,
	body io.Reader,
	h http.Header,
) ([]byte, int, error) {
	if url == "" {
		return nil, 0, errors.New("failed to fetch: missing url")
	}
	if method == "" {
		return nil, 0, errors.New("failed to fetch: missing method")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create the request: url=%s err=%v", url, err)
	}
	req.Header = h.Clone()

	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to readAll: %v", err)
	}

	return b, resp.StatusCode, nil
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
