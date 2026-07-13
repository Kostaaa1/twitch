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

	if h != nil {
		req.Header = h.Clone()
	}

	return c.Do(req)
}

func DoBytes(
	ctx context.Context,
	c *http.Client,
	url string,
	method string,
	body io.Reader,
	h http.Header,
) ([]byte, int, error) {
	resp, err := Do(ctx, c, url, method, body, h)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, fmt.Errorf("failed to read the error response: %v", err)
		}
		return b, resp.StatusCode, fmt.Errorf("invalid status %d: %s", resp.StatusCode, string(b))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to readAll: %v", err)
	}

	return b, resp.StatusCode, nil
}

func DoJSON(
	ctx context.Context,
	c *http.Client,
	url string,
	method string,
	body io.Reader,
	dst any,
	h http.Header,
) error {
	if dst == nil {
		return errors.New("dst must not be nit")
	}

	resp, err := Do(ctx, c, url, method, body, h)
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

	err = json.NewDecoder(resp.Body).Decode(dst)
	// drain leftover so that connection can be returned tu the pool and become reusable again
	io.Copy(io.Discard, resp.Body)
	return err
}
