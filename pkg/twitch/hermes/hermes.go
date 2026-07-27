package hermes

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	hermesURL = "wss://hermes.twitch.tv/v1?clientId=kimne78kx3ncx6brgo4mv6wki5h1ko"
)

type customType int

const (
	welcomeType customType = iota
	keepaliveType
	viewcountType
)

type Client struct {
	close    chan struct{}
	conn     *websocket.Conn
	connData *Welcome
}

type Subscribe struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"`
	PubSub *PubSub `json:"pubsub,omitempty"`
}

type SubscribeFrame struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Subscribe Subscribe `json:"subscribe"`
}

type Welcome struct {
	KeepaliveSec int    `json:"keepaliveSec,omitempty"`
	RecoveryURL  string `json:"recoveryUrl,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
}

type SubscribeResponse struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	PubSub       *PubSub `json:"pubsub"`
	Result       string  `json:"result,omitempty"`
	Subscription struct {
		ID string `json:"id,omitempty"`
	} `json:"subscription,omitempty"`
}

type ViewCountMessage struct {
	Type                 string  `json:"type,omitempty"`
	ServerTime           float64 `json:"server_time,omitempty"`
	Viewers              int     `json:"viewers,omitempty"`
	CollaborationStatus  string  `json:"collaboration_status,omitempty"`
	CollaborationViewers int     `json:"collaboration_viewers,omitempty"`
	CostreamStatus       string  `json:"costream_status,omitempty"`
	CostreamViewers      int     `json:"costream_viewers,omitempty"`
}

type PubSub struct {
	Topic string `json:"topic,omitempty"`
}

type SubNotification struct {
	ID           string `json:"id,omitempty"`
	Type         string `json:"type,omitempty"`
	Subscription *struct {
		ID string `json:"id"`
	} `json:"subscription,omitempty"`
	PubSub *PubSub `json:"pubsub,omitempty"`
}

type HermesResponse struct {
	ID                string           `json:"id,omitempty"`
	Timestamp         time.Time        `json:"timestamp,omitempty"`
	Type              string           `json:"type,omitempty"`
	ParentID          *string          `json:"parentId,omitempty"`
	Welcome           *Welcome         `json:"welcome,omitempty"`
	SubscribeResponse *SubNotification `json:"subscribeResponse,omitempty"`
	Notification      *SubNotification `json:"notification,omitempty"`
}

func generateID() string {
	b := make([]byte, 21)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (c *Client) Listen(ctx context.Context) error {
	fmt.Println("reading from WSS HERMES TWITCH")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.close:
			return nil
		default:
			var data interface{}
			if err := c.conn.ReadJSON(&data); err != nil {
				return err
			}
			b, err := json.MarshalIndent(data, "", " ")
			if err != nil {
				return err
			}
			fmt.Println("Hermes data: ", string(b))
		}
	}
}

func Dial(ctx context.Context) (*Client, error) {
	conn, resp, err := websocket.DefaultDialer.DialContext(
		ctx,
		hermesURL,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 101 {
		return nil, errors.New("status code not 101")
	}

	for k, v := range resp.Header {
		fmt.Printf("%s: %s\n", k, v)
	}

	return &Client{
		conn:  conn,
		close: make(chan struct{}, 1),
	}, nil
}

func ViewcountFrame(userID string) SubscribeFrame {
	return SubscribeFrame{
		ID:        generateID(),
		Type:      "subscribe",
		Timestamp: time.Now(),
		Subscribe: Subscribe{
			ID:   generateID(),
			Type: "pubsub",
			PubSub: &PubSub{
				Topic: fmt.Sprintf("video-playback-by-id.%s", userID),
			},
		},
	}
}

func (c *Client) Close() error {
	c.close <- struct{}{}
	return c.conn.Close()
}

func (c *Client) Subscribe(v interface{}) error {
	return c.conn.WriteJSON(v)
}

func (c *Client) Unsubscribe(ctx context.Context) {}
