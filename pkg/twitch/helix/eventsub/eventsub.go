package eventsub

import (
	"context"
	"fmt"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type EventSubMessage struct {
	Metadata Metadata `json:"metadata"`
	Payload  Payload  `json:"payload"`
}

type Metadata struct {
	MessageID           string    `json:"message_id"`
	MessageType         string    `json:"message_type"`
	MessageTimestamp    time.Time `json:"message_timestamp"`
	SubscriptionType    subType   `json:"subscription_type"`
	SubscriptionVersion string    `json:"subscription_version"`
}

type Payload struct {
	Session      *Session           `json:"session,omitempty"`
	Subscription *Subscription      `json:"subscription"`
	Event        *NotificationEvent `json:"event"`
}

type NotificationEvent struct {
	UserID               string    `json:"user_id"`
	UserLogin            string    `json:"user_login"`
	UserName             string    `json:"user_name"`
	BroadcasterUserID    string    `json:"broadcaster_user_id"`
	BroadcasterUserLogin string    `json:"broadcaster_user_login"`
	BroadcasterUserName  string    `json:"broadcaster_user_name"`
	FollowedAt           time.Time `json:"followed_at"`
	ID                   string    `json:"id"`
	Type                 string    `json:"type"`
	StartedAt            string    `json:"started_at"`
	ModeratorUserID      string    `json:"moderator_user_id"`
	ModeratorUserLogin   string    `json:"moderator_user_login"`
	ModeratorUserName    string    `json:"moderator_user_name"`
	MessageID            string    `json:"message_id"`
	Message              string    `json:"message"`
	Level                int       `json:"level"`
	Category             string    `json:"category"`
	Status               string    `json:"status"`
	HeldAt               time.Time `json:"held_at"`
	Fragments            struct {
		Emotes []struct {
			Text  string `json:"text"`
			ID    string `json:"id"`
			SetID string `json:"set-id"`
		} `json:"emotes"`
		Cheermotes []struct {
			Text   string `json:"text"`
			Amount int    `json:"amount"`
			Prefix string `json:"prefix"`
			Tier   int    `json:"tier"`
		} `json:"cheermotes"`
	} `json:"fragments"`
	TargetUserID    string `json:"target_user_id"`
	TargetUserName  string `json:"target_user_name"`
	TargetUserLogin string `json:"target_user_login"`
	// Message struct {
	// 	Text      string `json:"text"`
	// 	Fragments []struct {
	// 		Type      string      `json:"type"`
	// 		Text      string      `json:"text"`
	// 		Cheermote interface{} `json:"cheermote"`
	// 		Emote     interface{} `json:"emote"`
	// 	} `json:"fragments"`
	// } `json:"message"`
	Action      string      `json:"action"`
	FromAutomod bool        `json:"from_automod"`
	Terms       []string    `json:"terms"`
	Reason      string      `json:"reason"`
	Automod     interface{} `json:"automod"`
	BlockedTerm struct {
		TermsFound []struct {
			TermID                    string `json:"term_id"`
			OwnerBroadcasterUserID    string `json:"owner_broadcaster_user_id"`
			OwnerBroadcasterUserLogin string `json:"owner_broadcaster_user_login"`
			OwnerBroadcasterUserName  string `json:"owner_broadcaster_user_name"`
			Boundary                  struct {
				StartPos int `json:"start_pos"`
				EndPos   int `json:"end_pos"`
			} `json:"boundary"`
		} `json:"terms_found"`
	} `json:"blocked_term"`
}

type Session struct {
	ID                      string  `json:"id"`
	Status                  string  `json:"status"`
	ConnectedAt             string  `json:"connected_at"`
	KeepaliveTimeoutSeconds int     `json:"keepalive_timeout_seconds"`
	ReconnectURL            *string `json:"reconnect_url"`
	RecoveryURL             *string `json:"recovery_url"`
}

type Eventsub struct {
	client    *helix.Client
	transport transport
	g         *errgroup.Group
}

type WebsocketConnArgs struct {
	KeepaliveSeconds int
	OnNotification   func(n EventSubMessage)
	Events           []Event
}

func (e *Eventsub) Wait() error {
	return e.g.Wait()
}

func WithWebsocket(ctx context.Context, client *helix.Client, args WebsocketConnArgs) (*Eventsub, error) {
	url := "wss://eventsub.wss.twitch.tv/ws"
	if args.KeepaliveSeconds > 0 {
		url = fmt.Sprintf("%s?keepalive_timeout_seconds=%d", url, args.KeepaliveSeconds)
	}

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	_ = resp

	readyCh := make(chan struct{}, 1)

	g, ctx := errgroup.WithContext(ctx)

	e := &Eventsub{
		client: client,
		g:      g,
		transport: transport{
			Method: Websocket,
		},
	}

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				// fmt.Println("closing eventsub websocket goroutine")
				return conn.Close()
			default:
				var data EventSubMessage
				if err := conn.ReadJSON(&data); err != nil {
					return err
				}

				// fmt.Println("Websocket message:", data)

				switch data.Metadata.MessageType {
				case "session_keepalive":

				case "session_welcome":
					e.transport.SessionID = data.Payload.Session.ID
					readyCh <- struct{}{}
				case "reconnect":
					// fmt.Println("On reconnect caleld")
				case "notification":
					if args.OnNotification != nil {
						args.OnNotification(data)
					}
				}
			}
		}
	})

	<-readyCh

	return e, nil
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

func (e *Eventsub) StreamOnlineEvent(broadcasterID string) Event {
	return Event{
		Type:    "stream.online",
		Version: "1",
		Condition: Condition{
			"broadcaster_user_id": broadcasterID,
		},
		Transport: e.transport,
	}
}

func (e *Eventsub) StreamOfflineEvent(broadcasterID string) Event {
	return Event{
		Type:    "stream.offline",
		Version: "1",
		Condition: Condition{
			"broadcaster_user_id": broadcasterID,
		},
		Transport: e.transport,
	}
}

////// Pozdrav, potrebni su mi podaci za kontne planove posto ih nema u access-u. Isto mi je potreban odnos, sta se vezuje za kontne planove, vrsta naloga / usluga / nesto drugo?
////// U stvari, bolje je da sastavim detaljnije pitanje preko cc-a
//////
