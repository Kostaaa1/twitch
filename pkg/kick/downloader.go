package kick

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
)

type Progress struct {
	ID    string
	Label string
	Bytes int64
	Error error
	Done  bool
	Total float64
}

type Downloader struct {
	ctx        context.Context
	cycletls   cycletls.CycleTLS
	httpClient *http.Client
	notifyFn   func(Progress)
}

func New() *Downloader {
	return &Downloader{
		cycletls:   cycletls.Init(),
		httpClient: http.DefaultClient,
	}
}

func (c *Downloader) SetProgressNotifier(fn func(Progress)) {
	c.notifyFn = fn
}

func (c *Downloader) notify(unit *Unit, n int64) {
	if c.notifyFn == nil {
		return
	}
	c.notifyFn(Progress{
		ID:    unit.UUID.String(),
		Label: unit.GetLabel(),
		Error: unit.GetError(),
		Total: 0,
		Bytes: n,
		Done:  false,
	})
}

func (c *Downloader) Close() {
	c.cycletls.Close()
}

func (c *Downloader) defaultCycleTLSOpts() cycletls.Options {
	return cycletls.Options{
		Ja3:                   "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		UserAgent:             "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36",
		EnableConnectionReuse: true,
	}
}

func (c *Downloader) sendRequestAndDecode(URL string, method string, target interface{}) error {
	resp, err := c.cycletls.Do(URL, c.defaultCycleTLSOpts(), method)
	if err != nil {
		return err
	}
	return json.NewDecoder(strings.NewReader(resp.Body)).Decode(target)
}
