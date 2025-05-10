package twitch

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	Conn        *websocket.Conn
	Username    string
	Channels    []string
	AccessToken string
	ch          chan interface{}
}

var (
	re = regexp.MustCompile(`\b(PING|PRIVMSG|ROOMSTATE|USERNOTICE|USERSTATE|NOTICE|GLOBALUSERSTATE|CLEARMSG|CLEARCHAT)\b`)

	ErrAuthFailed         = errors.New("login authentication failed")
	ErrAuthImproperFormat = errors.New("improperly formatted auth")
)

func DialWS(name, token string, channels []string) (*WSClient, error) {
	socketURL := "ws://irc-ws.chat.twitch.tv:80"
	conn, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		return nil, err
	}
	return &WSClient{
		Conn:        conn,
		Username:    name,
		AccessToken: token,
		Channels:    channels,
	}, nil
}

func (c *WSClient) SetMessageChan(ch chan interface{}) {
	c.ch = ch
}

func (c *WSClient) SendMessage(msg []byte) error {
	return c.Conn.WriteMessage(websocket.TextMessage, msg)
}

func (c *WSClient) FormatIRCMsgAndSend(tag, channel, msg string) error {
	formatted := fmt.Sprintf("%s #%s :%s", tag, channel, msg)
	return c.SendMessage([]byte(formatted))
}

func (c *WSClient) LeaveChannel(channel string) {
	part := fmt.Sprintf("PART #%s", channel)
	c.SendMessage([]byte(part))
}

func (c *WSClient) ConnectToChannel(channel string) {
	join := fmt.Sprintf("JOIN #%s", channel)
	c.SendMessage([]byte(join))
}

func (c *WSClient) writeToChannel(msg interface{}) {
	if c.ch == nil {
		return
	}
	c.ch <- msg
}

func (c *WSClient) Connect() error {
	c.SendMessage([]byte("CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands"))

	pass := fmt.Sprintf("PASS oauth:%s", c.AccessToken)
	c.SendMessage([]byte(pass))

	nick := fmt.Sprintf("NICK %s", c.Username)
	c.SendMessage([]byte(nick))

	join := fmt.Sprintf("JOIN #%s", strings.Join(c.Channels, ",#"))
	c.SendMessage([]byte(join))

	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			return err
		}

		if msgType == websocket.TextMessage {
			rawIRCMessage := strings.TrimSpace(string(msg))
			tags := re.FindStringSubmatch(rawIRCMessage)

			if len(tags) > 1 {
				tag := tags[1]
				switch tag {
				case "USERSTATE":
					msg := parseROOMSTATE(rawIRCMessage)
					c.writeToChannel(msg)
				case "PRIVMSG":
					msg := parsePRIVMSG(rawIRCMessage)
					c.writeToChannel(msg)
				case "USERNOTICE":
					fmt.Println("USERNOTICE")
					parseUSERNOTICE(rawIRCMessage, c.ch)
				case "PING":
					c.SendMessage([]byte("PONG :tmi.twitch.tv"))
				case "NOTICE":
					msg := parseNOTICE(rawIRCMessage)
					if msg.SystemMsg == "Login authentication failed" {
						return ErrAuthFailed
					}
					c.writeToChannel(msg)
				}
			}
		}
	}
}
