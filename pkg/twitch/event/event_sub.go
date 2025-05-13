package event

import (
	"context"
	"fmt"
	"log"
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
		MessageID           string    `json:"message_id"`
		MessageType         string    `json:"message_type"`
		MessageTimestamp    time.Time `json:"message_timestamp"`
		SubscriptionType    string    `json:"subscription_type,omitempty"`
		SubscriptionVersion string    `json:"subscription_version,omitempty"`
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
		Subscription *Subscription `json:"subscription,omitempty"`
		// this can be anything??
		Event *struct {
			UserID               string    `json:"user_id"`
			UserLogin            string    `json:"user_login"`
			UserName             string    `json:"user_name"`
			BroadcasterUserID    string    `json:"broadcaster_user_id"`
			BroadcasterUserLogin string    `json:"broadcaster_user_login"`
			BroadcasterUserName  string    `json:"broadcaster_user_name"`
			FollowedAt           time.Time `json:"followed_at"`
		} `json:"event,omitempty"`
	} `json:"payload"`
}

type EventSub struct {
	tw            *twitch.Client
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

func (sub *EventSub) DialWSS(events []Event, keepalive time.Duration) error {
	url := fmt.Sprintf("wss://eventsub.wss.twitch.tv/ws?keepalive_timeout_seconds=%d", int(keepalive.Seconds()))

	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("failed to dial eventsub.wss: %v", err)
	}
	defer resp.Body.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		go func() {
			if err := sub.DeleteAllSubscriptions(ctx); err != nil {
				fmt.Printf("failed to delete all subscriptions: %v\n", err)
			}
			close(done)
		}()

		select {
		case <-done:
			fmt.Println("Deleted all subscriptions")
		}

		fmt.Println("Closing connection!")
		cancel()
		conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Connection closed!")
			return nil
		default:
			var msg WebsocketConnResponse
			if err := conn.ReadJSON(&msg); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				fmt.Printf("error while reading the json msg: %v\n", err)
				continue
			}

			switch MessageType(msg.Metadata.MessageType) {
			case Revocation:
				fmt.Println("revocation message:", msg)
			case Notification:
				fmt.Println("notification message:", msg)
			case KeepAlive:
				fmt.Println("keepalive message:", msg)
			case SessionWelcome:
				transport := WebsocketTransport(msg.Payload.Session.ID)
				for _, event := range events {
					body := RequestBody{
						Version:   event.Version,
						Condition: event.Condition,
						Type:      event.Type,
						Transport: transport,
					}
					resp, err := sub.Subscribe(body)
					if err != nil {
						log.Fatal(err)
					}
					subData := resp.Data
					fmt.Println("Successfully subscribed:", subData[0].ID)
				}
			}
		}
	}
}
