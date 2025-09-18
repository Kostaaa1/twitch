package kick

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
)

type ProgressMessage struct {
	ID    any
	Bytes int64
	Error error
	Done  bool
}

type Client struct {
	ctx        context.Context
	cycletls   cycletls.CycleTLS
	httpClient *http.Client
	notifyFn   func(ProgressMessage)
}

func New() *Client {
	return &Client{
		cycletls:   cycletls.Init(),
		httpClient: http.DefaultClient,
	}
}

func (c *Client) SetProgressNotifier(fn func(ProgressMessage)) {
	c.notifyFn = fn
}

func (c *Client) notify(msg ProgressMessage) {
	if c.notifyFn != nil {
		c.notifyFn(msg)
	}
}

func (c *Client) Close() {
	c.cycletls.Close()
}

func (c *Client) defaultCycleTLSOpts() cycletls.Options {
	return cycletls.Options{
		Ja3:       "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36",
	}
}

func (c *Client) sendRequestAndDecode(URL string, method string, target interface{}) error {
	resp, err := c.cycletls.Do(URL, c.defaultCycleTLSOpts(), method)
	if err != nil {
		return err
	}
	return json.NewDecoder(strings.NewReader(resp.Body)).Decode(target)
}
