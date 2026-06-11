package eventsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/gorilla/websocket"
)

type websocketEventsub struct {
	socketID string
	conn     *websocket.Conn
}

type Eventsub struct {
	client    *helix.Client
	transport Transport
	wss       *websocketEventsub
	active    *EventsubResponse[Subscription]
}

type EventSubMessage struct {
	Metadata Metadata `json:"metadata"`
	Payload  Payload  `json:"payload"`
}

type Metadata struct {
	MessageID        string `json:"message_id"`
	MessageType      string `json:"message_type"`
	MessageTimestamp string `json:"message_timestamp"`
}

type Payload struct {
	Session *Session `json:"session,omitempty"`
}

type Session struct {
	ID                      string  `json:"id"`
	Status                  string  `json:"status"`
	ConnectedAt             string  `json:"connected_at"`
	KeepaliveTimeoutSeconds int     `json:"keepalive_timeout_seconds"`
	ReconnectURL            *string `json:"reconnect_url"`
	RecoveryURL             *string `json:"recovery_url"`
}

func New(c *helix.Client, t Transport) *Eventsub {
	e := &Eventsub{
		client:    c,
		transport: t,
	}

	switch t.Method {
	case Websocket:
		ctx := context.Background()

		url := "wss://eventsub.wss.twitch.tv/ws?keepalive_timeout_seconds=30"

		conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
		if err != nil {
			log.Fatal(err)
		}

		for {
			var data EventSubMessage
			if err := conn.ReadJSON(&data); err != nil {
				log.Fatal(err)
			}

			switch data.Metadata.MessageType {
			case "session_keepalive":
			case "session_welcome":
				e.transport.SessionID = data.Payload.Session.ID

				user, err := c.Users().UserLogin("slorpglorpski").Run(ctx)
				if err != nil {
					log.Fatal(err)
				}

				userID := user.Data[0].ID
				_ = userID

				// 1
				// event := e.StreamOnlineEvent(userID)
				// resp, err := e.Subscriptions().Create(event).Run(ctx)
				// if err != nil {
				// 	log.Fatal(err)
				// }
				// fmt.Println("Created:", resp)

				// 2
				// event = e.StreamOfflineEvent(userID)
				// resp, err = e.Subscriptions().Create(event).Run(ctx)
				// if err != nil {
				// 	log.Fatal(err)
				// }

				subs, err := e.Subscriptions().Get().Run(ctx)
				if err != nil {
					log.Fatal(err)
				}

				b, _ := json.MarshalIndent(subs, "", " ")
				fmt.Println("Waiting... Subscriptions:", string(b))

			case "reconnect":
				// reconnect
			case "notification":
				b, _ := json.MarshalIndent(data, "", " ")
				fmt.Println("NOTIFICATION:")
				fmt.Println(string(b))
			}
		}

	case Webhook:
		if t.Callback == "" {
			return nil
		}
		t.Secret = "testtesttesttest"
	}

	return e
}

type EventsubResponse[T any] struct {
	Total        int `json:"total"`
	Data         []T `json:"data"`
	TotalCost    int `json:"total_cost"`
	MaxTotalCost int `json:"max_total_cost"`
	Pagination   struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

func (e *Eventsub) StreamOnlineEvent(broadcasterID string) *Event {
	return &Event{
		Type:    "stream.online",
		Version: "1",
		Condition: Condition{
			"broadcaster_user_id": broadcasterID,
		},
		Transport: e.transport,
	}
}

func (e *Eventsub) StreamOfflineEvent(broadcasterID string) *Event {
	return &Event{
		Type:    "stream.offline",
		Version: "1",
		Condition: Condition{
			"broadcaster_user_id": broadcasterID,
		},
		Transport: e.transport,
	}
}
