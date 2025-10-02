package eventsub

import (
	"context"
	"fmt"
	"sync"
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
	tw             *twitch.Client
	Subscriptions  []SubscriptionMessage
	OnRevocation   func(resp ResponseBody)
	OnReconnect    func(resp ResponseBody)
	OnKeepAlive    func(resp ResponseBody)
	OnNotification func(resp ResponseBody)
	mu             sync.Mutex
}

func New(tw *twitch.Client) *EventSubClient {
	return &EventSubClient{
		tw:            tw,
		Subscriptions: []SubscriptionMessage{},
	}
}

func (client *EventSubClient) DialWS(ctx context.Context, events []Event) error {
	// subs, _ := client.GetSubscriptions()
	// if len(subs.Data) > 0 {
	// 	fmt.Println("found: ", len(subs.Data))
	// 	client.Subscriptions = subs.Data
	// 	client.UnsubscribeToAll()
	// }

	conn, resp, err := websocket.DefaultDialer.Dial("wss://eventsub.wss.twitch.tv/ws?keepalive_timeout_seconds=10", nil)
	if err != nil {
		return fmt.Errorf("failed to dial eventsub.wss: %v", err)
	}
	defer resp.Body.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Println("Connected to eventsub websocket!")

	msgchan := make(chan ResponseBody)
	errChan := make(chan error)

	// read the conn messages in separate goroutine, because conn.ReadJSON is blocking call which means that select {case <-ctx.Done()} will never run until conn.ReadJSON finished and we need to simultaneously wait for ctx.Done message. So this is the pattern that I should recognize, i can avoid blocking with spawning blocking code in goroutine which will communicate via channels
	go func() {
		for {
			var msg ResponseBody
			err := conn.ReadJSON(&msg)
			if err != nil {
				errChan <- err
				return
			}
			msgchan <- msg
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Unsubscribing to the events...")
			if err := client.UnsubscribeToAll(ctx); err != nil {
				fmt.Printf("failed to delete all subscriptions: %v\n", err)
			}
			fmt.Println("Closing the connection...")
			conn.Close()
			return nil
		case err := <-errChan:
			fmt.Println(err)
		case msg := <-msgchan:
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
				fmt.Println("Keepalive message!")
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
					resp, err := client.Subscribe(ctx, body)
					if err != nil {
						fmt.Printf("Failed to subscribe to: %s | error: %s\n", event.Type, err)
						continue
					}
					subData := resp.Data
					fmt.Printf("Subscribed: %s | event_id: %s\n", event.Type, subData[0].ID)
				}
			}
		}
	}
}
