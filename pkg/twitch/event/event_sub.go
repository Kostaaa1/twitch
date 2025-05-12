package event

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/gorilla/websocket"
)

type MessageType string

var (
	SessionWelcome MessageType = "session_welcome"
	Notification   MessageType = "notification"
	Revocation     MessageType = "revocation"
	KeepAlive      MessageType = "session_keepalive"
	Reconnect      MessageType = "session_reconnect"
)

type WebsocketConnResponse struct {
	Metadata struct {
		MessageID        string    `json:"message_id"`
		MessageType      string    `json:"message_type"`
		MessageTimestamp time.Time `json:"message_timestamp"`
	} `json:"metadata"`
	Payload struct {
		Session *struct {
			ID                      string    `json:"id"`
			Status                  string    `json:"status"`
			ConnectedAt             time.Time `json:"connected_at"`
			KeepaliveTimeoutSeconds int       `json:"keepalive_timeout_seconds"`
			ReconnectURL            any       `json:"reconnect_url"`
			RecoveryURL             any       `json:"recovery_url"`
		} `json:"session,omitempty"`
	} `json:"payload"`
}

type Subscription struct {
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

type EventSub struct {
	tw            *twitch.Client
	SessionID     string
	Subscriptions []Subscription
	Total         int
	TotalCost     int
	MaxTotalCost  int
}

func NewSub(tw *twitch.Client) *EventSub {
	return &EventSub{
		tw:            tw,
		Subscriptions: []Subscription{},
		Total:         0,
		TotalCost:     0,
		MaxTotalCost:  0,
	}
}

func (sub *EventSub) DialWSS(events []Event) error {
	sub.GetSubscriptions()
	return nil

	eventsub := "wss://eventsub.wss.twitch.tv/ws?keepalive_timeout_seconds=10"
	conn, resp, err := websocket.DefaultDialer.Dial(eventsub, nil)
	if err != nil {
		return fmt.Errorf("failed to dial eventsub.wss: %v", err)
	}
	defer resp.Body.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		conn.Close()
		os.Exit(1)
	}()

	for {
		var msg WebsocketConnResponse
		if err := conn.ReadJSON(&msg); err != nil {
			fmt.Printf("error while reading the json msg: %v\n", err)
			continue
		}

		switch MessageType(msg.Metadata.MessageType) {
		case Revocation:
		case Notification:
		case KeepAlive:
		case SessionWelcome:
		}

		// if msg.Payload.Session.Status != "connected" {
		// 	fmt.Println("not connected: ", msg)
		// 	continue
		// }

		// sub.SessionID = msg.Payload.Session.ID
		// transport := WebsocketTransport(msg.Payload.Session.ID)

		// if len(sub.Subscriptions) == 0 {
		// 	for _, event := range events {
		// 		body := RequestBody{
		// 			Version:   event.Version,
		// 			Condition: event.Condition,
		// 			Type:      "stream.online",
		// 			Transport: transport,
		// 		}
		// 		resp, err := sub.Subscribe(body)
		// 		if err != nil {
		// 			log.Fatal(err)
		// 		}
		// 		fmt.Println("Subscription response: ", resp)
		// 	}
		// }
	}
}
