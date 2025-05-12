package event

import (
	"context"
	"encoding/json"
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

func (sub *EventSub) DialWSS(events []Event) error {
	eventsub := "wss://eventsub.wss.twitch.tv/ws?keepalive_timeout_seconds=10"

	conn, resp, err := websocket.DefaultDialer.Dial(eventsub, nil)
	if err != nil {
		return fmt.Errorf("failed to dial eventsub.wss: %v", err)
	}
	defer resp.Body.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("deleting active subscriptions... found: ", sub.Total)
		if err := sub.DeleteAllSubscriptions(); err != nil {
			fmt.Printf("failed to delete all subscriptions: %v\n", err)
		}
		cancel()
		fmt.Println("closing connection...")
		conn.Close()
	}()

	test, _ := sub.GetSubscriptions()
	fmt.Println("LENGTH: ", len(test.Data))

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Connection closed!")
			return nil
		default:
			var msg WebsocketConnResponse
			if err := conn.ReadJSON(&msg); err != nil {
				fmt.Printf("error while reading the json msg: %v\n", err)
				continue
			}

			switch MessageType(msg.Metadata.MessageType) {
			case Revocation:
				fmt.Println("revocation message: ", msg)
			case Notification:
				fmt.Println("notification message: ", msg)
			case KeepAlive:
				fmt.Println("keepalive message: ", msg)
			case SessionWelcome:
				transport := WebsocketTransport(msg.Payload.Session.ID)
				for _, event := range events {
					body := RequestBody{
						Version:   event.Version,
						Condition: event.Condition,
						Type:      event.Type,
						Transport: transport,
					}
					_, err := sub.Subscribe(body)
					if err != nil {
						log.Fatal(err)
					}
				}

				resp, err := sub.GetSubscriptions()
				if err != nil {
					fmt.Printf("failed to get subscriptions: %v\n", err)
					continue
				}

				b1, _ := json.MarshalIndent(resp.Data, "", " ")
				b2, _ := json.MarshalIndent(sub.Subscriptions, "", " ")

				fmt.Println("B1: ", string(b1))
				fmt.Println("B2: ", string(b2))
				time.Sleep(8 * time.Second)

				resp, err = sub.GetSubscriptions()
				if err != nil {
					fmt.Printf("failed to get subscriptions: %v\n", err)
					continue
				}
				b3, _ := json.MarshalIndent(resp.Data, "", " ")
				fmt.Println("B3: ", string(b3))

				// sub.Subscriptions = resp.Data
				// sub.MaxTotalCost = resp.MaxTotalCost
				// sub.TotalCost = resp.TotalCost
				// sub.Total = resp.Total
			}
		}
	}
}
