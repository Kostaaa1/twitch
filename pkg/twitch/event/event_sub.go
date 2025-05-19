package event

import (
	"context"
	"fmt"
	"log"
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

type RequestBody struct {
	Version   int32                  `json:"version"`
	Type      Type                   `json:"type"`
	Condition map[string]interface{} `json:"condition"`
	Transport Transport              `json:"transport"`
}

type ResponseBody struct {
	Metadata MessageMetadata `json:"metadata"`
	Payload  struct {
		Session      *SessionMessage      `json:"session,omitempty"`
		Subscription *SubscriptionMessage `json:"subscription,omitempty"`
		Event        *EventMessage        `json:"event,omitempty"`
	} `json:"payload"`
}

type SessionMessage struct {
	ID                      string    `json:"id"`
	Status                  string    `json:"status"`
	ConnectedAt             time.Time `json:"connected_at"`
	KeepaliveTimeoutSeconds int       `json:"keepalive_timeout_seconds"`
	ReconnectURL            any       `json:"reconnect_url"`
	RecoveryURL             any       `json:"recovery_url"`
}

type MessageMetadata struct {
	MessageID           string    `json:"message_id"`
	MessageType         string    `json:"message_type"`
	MessageTimestamp    time.Time `json:"message_timestamp"`
	SubscriptionType    string    `json:"subscription_type,omitempty"`
	SubscriptionVersion string    `json:"subscription_version,omitempty"`
}

type EventMessage struct {
	UserID               string    `json:"user_id"`
	UserLogin            string    `json:"user_login"`
	UserName             string    `json:"user_name"`
	BroadcasterUserID    string    `json:"broadcaster_user_id"`
	BroadcasterUserLogin string    `json:"broadcaster_user_login"`
	BroadcasterUserName  string    `json:"broadcaster_user_name"`
	FollowedAt           time.Time `json:"followed_at"`
}

type EventSubClient struct {
	tw            *twitch.Client
	Subscriptions []SubscriptionMessage
	Total         int
	TotalCost     int
	MaxTotalCost  int

	OnRevocation   func(resp ResponseBody)
	OnReconnect    func(resp ResponseBody)
	OnKeepAlive    func(resp ResponseBody)
	OnNotification func(resp ResponseBody)
	// OnSessionWelcome func(resp ResponseBody)
}

func NewClient(tw *twitch.Client) *EventSubClient {
	return &EventSubClient{
		tw:            tw,
		Subscriptions: []SubscriptionMessage{},
	}
}

func (client *EventSubClient) DialWS(ctx context.Context, events []Event) error {
	conn, resp, err := websocket.DefaultDialer.Dial("wss://eventsub.wss.twitch.tv/ws?keepalive_timeout_seconds=30", nil)
	if err != nil {
		return fmt.Errorf("failed to dial eventsub.wss: %v", err)
	}
	defer resp.Body.Close()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Connection closed!")
			conn.Close()
			return nil
		default:
			var msg ResponseBody
			if err := conn.ReadJSON(&msg); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				fmt.Printf("error while reading the json msg: %v\n", err)
				continue
			}

			switch MessageType(msg.Metadata.MessageType) {
			case Revocation:
				if client.OnRevocation != nil {
					client.OnRevocation(msg)
				}
			case Notification:
				if client.OnNotification != nil {
					client.OnNotification(msg)
				}
			case KeepAlive:
				if client.OnKeepAlive != nil {
					client.OnKeepAlive(msg)
				}
			case Reconnect:
				if client.OnReconnect != nil {
					client.OnReconnect(msg)
				}
			case SessionWelcome:
				transport := WebsocketTransport(msg.Payload.Session.ID)
				for _, event := range events {
					body := RequestBody{
						Version:   event.Version,
						Condition: event.Condition,
						Type:      event.Type,
						Transport: transport,
					}

					resp, err := client.Subscribe(body)
					if err != nil {
						log.Fatal(err)
					}
					subData := resp.Data

					fmt.Printf("Subscription successful [event: %s | event_id: %s]\n", event.Type, subData[0].ID)
				}
			}
		}
	}
}

// ctx, cancel := context.WithCancel(context.Background())
// defer cancel()
// done := make(chan struct{})
// c := make(chan os.Signal, 1)
// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
// go func() {
// 	<-c
// 	fmt.Println("Closing...")
// 	go func() {
// 		if err := client.UnsubscribeToAll(); err != nil {
// 			fmt.Printf("failed to delete all subscriptions: %v\n", err)
// 		}
// 		close(done)
// 	}()
// 	select {
// 	case <-done:
// 		fmt.Println("Dispatched subscribed events!")
// 	}
// 	fmt.Println("Closing connection!")
// 	conn.Close()
// 	// cancel()
// }()
