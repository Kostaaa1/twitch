package kick

import (
	"context"
	"net"
	"net/http"

	utls "github.com/refraction-networking/utls"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := utls.Dial(network, addr, nil)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
	return &Client{
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

func (c *Client) setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0")
	// req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	// req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
}
