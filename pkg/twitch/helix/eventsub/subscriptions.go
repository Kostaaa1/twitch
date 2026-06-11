package eventsub

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
)

type Subscription struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Type      string                 `json:"type"`
	Version   string                 `json:"version"`
	Condition map[string]interface{} `json:"condition"`
	CreatedAt time.Time              `json:"created_at"`
	Transport Transport              `json:"transport"`
	Cost      int                    `json:"cost"`
}

type subscriptionCmd struct {
	c   *helix.Client
	url *url.URL
}

func (e *Eventsub) Subscriptions() *subscriptionCmd {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/eventsub/subscriptions")
	return &subscriptionCmd{
		c:   e.client,
		url: parsed,
	}
}

type subGetCmd struct {
	c      *helix.Client
	url    *url.URL
	values url.Values
}

func (s *subscriptionCmd) Get() *subGetCmd {
	return &subGetCmd{
		c:      s.c,
		url:    s.url,
		values: url.Values{},
	}
}

func (c *subGetCmd) Status(subStat SubStatus) *subGetCmd {
	c.values.Add("status", string(subStat))
	return c
}
func (c *subGetCmd) Type(subType string) *subGetCmd {
	c.values.Add("type", subType)
	return c
}
func (c *subGetCmd) UserID(userID string) *subGetCmd {
	c.values.Add("user_id", userID)
	return c
}
func (c *subGetCmd) SubscriptionID(subID string) *subGetCmd {
	c.values.Add("subscription_id", subID)
	return c
}
func (c *subGetCmd) ConduitID(conduitID string) *subGetCmd {
	c.values.Add("conduit_id", conduitID)
	return c
}
func (c *subGetCmd) After(cursor string) *subGetCmd {
	c.values.Add("after", cursor)
	return c
}

func (cmd *subGetCmd) Run(ctx context.Context) (*EventsubResponse[Subscription], error) {
	cmd.url.RawQuery = cmd.values.Encode()

	var data EventsubResponse[Subscription]

	if err := cmd.c.Request(
		ctx,
		cmd.url.String(),
		http.MethodGet,
		nil,
		&data,
	); err != nil {
		return nil, err
	}

	return &data, nil
}

type subDeleteCmd struct {
	c      *helix.Client
	url    *url.URL
	values url.Values
}

func (s *subscriptionCmd) Delete(id string) *subDeleteCmd {
	return &subDeleteCmd{
		c:      s.c,
		url:    s.url,
		values: url.Values{"id": {id}},
	}
}

func (cmd *subDeleteCmd) Run(ctx context.Context) error {
	cmd.url.RawQuery = cmd.values.Encode()
	return cmd.c.Request(
		ctx,
		cmd.url.String(),
		http.MethodDelete,
		nil,
		nil,
	)
}

type CreateSubscriptionResponse struct {
	ID        string            `json:"id"`
	Status    string            `json:"status"`
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	Condition map[string]string `json:"condition"`
	CreatedAt time.Time         `json:"created_at"`
	Transport struct {
		Method   string `json:"method"`
		Callback string `json:"callback"`
	} `json:"transport"`
	Cost int `json:"cost"`
}

type subCreateCmd struct {
	c   *helix.Client
	url *url.URL
	e   *Event
}

func (s *subscriptionCmd) Create(e *Event) *subCreateCmd {
	return &subCreateCmd{
		c:   s.c,
		url: s.url,
		e:   e,
	}
}

func (cmd *subCreateCmd) Run(ctx context.Context) (*EventsubResponse[CreateSubscriptionResponse], error) {
	if err := cmd.e.Validate(); err != nil {
		return nil, err
	}

	b, err := json.Marshal(cmd.e)
	if err != nil {
		return nil, err
	}

	var data EventsubResponse[CreateSubscriptionResponse]

	if err := cmd.c.Request(
		ctx,
		cmd.url.String(),
		http.MethodPost,
		bytes.NewReader(b),
		&data,
	); err != nil {
		return nil, err
	}

	return &data, nil
}
