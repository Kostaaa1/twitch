package helix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Eventsub struct {
	client   *Client
	secret   string
	callback string
}

func New(c *Client) *Eventsub {
	secret := "WUlJg1t8WpDC98gl6K4lGxryJrCeqOGX"
	callback := "https://example.com/webhooks/callback"
	return &Eventsub{c, secret, callback}
}

func (e *Eventsub) StreamOnlineEvent(broadcasterID string) *Event {
	return &Event{
		Type:    "stream.online",
		Version: "1",
		Condition: map[string]string{
			"broadcaster_id": broadcasterID,
		},
		Transport: WebhookTransport{
			Method:   "webhook",
			Callback: e.callback,
			Secret:   e.secret,
		},
	}
}

func (e *Eventsub) StreamOfflineEvent(broadcasterID string) *Event {
	return &Event{
		Type:    "stream.offline",
		Version: "1",
		Condition: map[string]string{
			"broadcaster_id": broadcasterID,
		},
		Transport: WebhookTransport{
			Method:   "webhook",
			Callback: e.callback,
			Secret:   e.secret,
		},
	}
}

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

type WebhookRequest struct {
	Data []struct {
		ID        string            `json:"id"`
		Status    string            `json:"status"`
		Type      string            `json:"type"`
		Version   string            `json:"version"`
		Cost      int               `json:"cost"`
		Condition map[string]string `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"data"`
	Total        int `json:"total"`
	TotalCost    int `json:"total_cost"`
	MaxTotalCost int `json:"max_total_cost"`
}

func (e *Eventsub) Subscribe(ctx context.Context, event *Event) error {
	u := "https://api.twitch.tv/helix/eventsub/subscriptions"

	b, err := json.Marshal(event)
	if err != nil {
		return err
	}

	var data interface{}
	if err := e.client.Request(ctx, u, http.MethodPost, bytes.NewReader(b), &data); err != nil {
		return err
	}

	fmt.Println("SUBSCIPRITON DATA:", data)

	return nil
}

func (e *Eventsub) Unsubscribe(ctx context.Context, subscriptionID string) error {
	u := "https://api.twitch.tv/helix/eventsub/subscriptions?id=SUBSCRIPTION_ID"
	var data interface{}
	if err := e.client.Request(ctx, u, http.MethodDelete, nil, &data); err != nil {
		return err
	}
	fmt.Println("UNSUBSCRIBED", data)
	return nil
}

func (e *Eventsub) Subscriptions(ctx context.Context) error {
	u := "https://api.twitch.tv/helix/eventsub/subscriptions"

	var data interface{}
	if err := e.client.Request(ctx, u, http.MethodGet, nil, &data); err != nil {
		return err
	}

	fmt.Println("SUBSCIPRITONS DATA:", data)

	return nil
}
