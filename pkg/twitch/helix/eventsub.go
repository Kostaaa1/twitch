package helix

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix/eventsub"
)

type Eventsub struct {
	client *Client
}

func NewEventsub(c *Client) *Eventsub {
	return &Eventsub{c}
}

type Subscriptions struct {
	Total int `json:"total"`
	Data  []struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
		} `json:"condition"`
		CreatedAt time.Time `json:"created_at"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		Cost int `json:"cost"`
	} `json:"data"`
	TotalCost    int `json:"total_cost"`
	MaxTotalCost int `json:"max_total_cost"`
	Pagination   struct {
	} `json:"pagination"`
}

type getSubscriptionsCmd struct {
	c      *Client
	url    *url.URL
	values url.Values
}

// filter subscriptions by its status
func (c *getSubscriptionsCmd) Status(subStat eventsub.SubStatus) *getSubscriptionsCmd {
	c.values.Add("status", string(subStat))
	return c
}

func (c *getSubscriptionsCmd) Type(subType string) *getSubscriptionsCmd {
	c.values.Add("type", subType)
	return c
}

func (c *getSubscriptionsCmd) UserID(userID string) *getSubscriptionsCmd {
	c.values.Add("user_id", userID)
	return c
}

func (c *getSubscriptionsCmd) SubscriptionID(subID string) *getSubscriptionsCmd {
	c.values.Add("subscription_id", subID)
	return c
}

func (c *getSubscriptionsCmd) ConduitID(conduitID string) *getSubscriptionsCmd {
	c.values.Add("conduit_id", conduitID)
	return c
}

func (c *getSubscriptionsCmd) After(cursor string) *getSubscriptionsCmd {
	c.values.Add("after", cursor)
	return c
}

func (c *getSubscriptionsCmd) Run(ctx context.Context) (*helixEnvelope[Subscriptions], error) {
	c.url.RawQuery = c.values.Encode()
	var body helixEnvelope[Subscriptions]
	if err := c.c.Request(ctx, c.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Eventsub) Subscriptions() *getSubscriptionsCmd {
	parsed, _ := url.Parse("https://api.twitch.tv/helix/eventsub/subscriptions")
	return &getSubscriptionsCmd{
		c:      c.client,
		url:    parsed,
		values: url.Values{},
	}
}

type deleteSubscriptionCmd struct {
	c      *Client
	url    *url.URL
	values url.Values
}

func (c *deleteSubscriptionCmd) Status(subStat eventsub.SubStatus) *deleteSubscriptionCmd {
	c.values.Add("status", string(subStat))
	return c
}

func (c *deleteSubscriptionCmd) Type(subType string) *deleteSubscriptionCmd {
	c.values.Add("type", subType)
	return c
}

func (c *deleteSubscriptionCmd) UserID(userID string) *deleteSubscriptionCmd {
	c.values.Add("user_id", userID)
	return c
}

func (c *deleteSubscriptionCmd) SubscriptionID(subID string) *deleteSubscriptionCmd {
	c.values.Add("subscription_id", subID)
	return c
}

func (c *deleteSubscriptionCmd) ConduitID(conduitID string) *deleteSubscriptionCmd {
	c.values.Add("conduit_id", conduitID)
	return c
}

func (c *deleteSubscriptionCmd) After(cursor string) *deleteSubscriptionCmd {
	c.values.Add("after", cursor)
	return c
}

func (c *deleteSubscriptionCmd) Run(ctx context.Context) (*helixEnvelope[Subscriptions], error) {
	c.url.RawQuery = c.values.Encode()
	var body helixEnvelope[Subscriptions]
	if err := c.c.Request(ctx, c.url.String(), http.MethodGet, nil, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *Eventsub) Subscribe(ctx context.Context, event *Event) (*helixEventsubEnvelope[CreateSubscriptionResponse], error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}

	var data helixEventsubEnvelope[CreateSubscriptionResponse]
	if err := c.client.Request(
		ctx,
		"https://api.twitch.tv/helix/eventsub/subscriptions",
		http.MethodPost,
		event,
		&data,
	); err != nil {
		return nil, err
	}

	return &data, nil
}

// func (e *Eventsub) StreamOnlineEvent(broadcasterID string) *Event {
// 	return &Event{
// 		Type:    "stream.online",
// 		Version: "1",
// 		Condition: map[string]string{
// 			"broadcaster_id": broadcasterID,
// 		},
// 		Transport: WebhookTransport{
// 			Method:   "webhook",
// 			Callback: e.callback,
// 			Secret:   e.secret,
// 		},
// 	}
// }

// func (e *Eventsub) StreamOfflineEvent(broadcasterID string) *Event {
// 	return &Event{
// 		Type:    "stream.offline",
// 		Version: "1",
// 		Condition: map[string]string{
// 			"broadcaster_id": broadcasterID,
// 		},
// 		Transport: WebhookTransport{
// 			Method:   "webhook",
// 			Callback: e.callback,
// 			Secret:   e.secret,
// 		},
// 	}
// }

// type Transport struct {
// 	Method string `json:"method"`
// 	// webhook
// 	Callback string `json:"callback"`
// 	Secret   string `json:"secret"`
// 	// websocket
// 	SessionID      string `json:"session_id"`
// 	ConnectedAt    string `json:"connected_at"`
// 	DisconnectedAt string `json:"disconnected_at"`
// }

type CreateSubscriptionResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	Condition struct {
		UserID string `json:"user_id"`
	} `json:"condition"`
	CreatedAt time.Time `json:"created_at"`
	Transport struct {
		Method   string `json:"method"`
		Callback string `json:"callback"`
	} `json:"transport"`
	Cost int `json:"cost"`
}

type WebhookTransport struct {
	Method   string `json:"method"`
	Callback string `json:"callback"`
	Secret   string `json:"secret"`
}

type Event struct {
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	Condition map[string]string `json:"condition"`
	Transport WebhookTransport  `json:"transport"`
}

func (e *Event) Validate() error {
	return nil
}

type Subscription[T any] struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	Status    string `json:"status"`
	Cost      int    `json:"cost"`
	Condition T      `json:"condition"`
	Transport struct {
		Method   string `json:"method"`
		Callback string `json:"callback"`
	} `json:"transport"`
	CreatedAt time.Time `json:"created_at"`
}

type Notification[T, K any] struct {
	Subscription Subscription[K] `json:"subscription"`
	Event        T               `json:"event"`
}

type StreamOnlineNotificationCondition struct {
	BroadcasterUserID string `json:"broadcaster_user_id"`
}

type StreamOnlineNotificationEvent struct {
	ID                   string    `json:"id"`
	BroadcasterUserID    string    `json:"broadcaster_user_id"`
	BroadcasterUserLogin string    `json:"broadcaster_user_login"`
	BroadcasterUserName  string    `json:"broadcaster_user_name"`
	Type                 string    `json:"type"`
	StartedAt            time.Time `json:"started_at"`
}

type StreamOnlineNotification Notification[StreamOnlineNotificationEvent, StreamOnlineNotificationCondition]

// type WebhookRequest struct {
// 	Data []struct {
// 		ID        string            `json:"id"`
// 		Status    string            `json:"status"`
// 		Type      string            `json:"type"`
// 		Version   string            `json:"version"`
// 		Cost      int               `json:"cost"`
// 		Condition map[string]string `json:"condition"`
// 		Transport struct {
// 			Method   string `json:"method"`
// 			Callback string `json:"callback"`
// 		} `json:"transport"`
// 		CreatedAt time.Time `json:"created_at"`
// 	} `json:"data"`
// 	Total        int `json:"total"`
// 	TotalCost    int `json:"total_cost"`
// 	MaxTotalCost int `json:"max_total_cost"`
// }
