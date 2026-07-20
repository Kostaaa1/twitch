package chat

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

type TwitchIRC struct {
	conn *websocket.Conn
	C    chan interface{}
}

var (
	re                    = regexp.MustCompile(`\b(PING|PRIVMSG|ROOMSTATE|USERNOTICE|USERSTATE|NOTICE|GLOBALUSERSTATE|CLEARMSG|CLEARCHAT)\b`)
	ErrInvalidNick        = errors.New("failed to join iirc: invalid nick")
	ErrAuthFailed         = errors.New("login authentication failed")
	ErrAuthImproperFormat = errors.New("improperly formatted auth")
)

func DialIRC(username, accessToken string, channels []string) (*TwitchIRC, error) {
	socketURL := "wss://irc-ws.chat.twitch.tv:443"

	conn, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		return nil, err
	}

	return &TwitchIRC{
		conn: conn,
		C:    make(chan interface{}),
	}, nil
}

func (c *TwitchIRC) Close() error {
	return c.conn.Close()
}

func (c *TwitchIRC) SendMessage(msg []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

func (c *TwitchIRC) FormatIRCMsgAndSend(tag, channel, msg string) error {
	formatted := fmt.Sprintf("%s #%s :%s", tag, channel, msg)
	return c.SendMessage([]byte(formatted))
}

func (c *TwitchIRC) LeaveChannel(channel string) {
	part := fmt.Sprintf("PART #%s", channel)
	c.SendMessage([]byte(part))
}

func (c *TwitchIRC) ConnectToChannel(channel string) {
	join := fmt.Sprintf("JOIN #%s", channel)
	c.SendMessage([]byte(join))
}

func (c *TwitchIRC) writeToChannel(msg interface{}) {
	if c.C == nil {
		return
	}
	c.C <- msg
}

func (c *TwitchIRC) Connect(ctx context.Context, accessToken, username string, channels []string) error {
	c.SendMessage([]byte("CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands"))

	pass := fmt.Sprintf("PASS oauth:%s", accessToken)
	c.SendMessage([]byte(pass))

	nick := fmt.Sprintf("NICK %s", username)
	c.SendMessage([]byte(nick))

	join := fmt.Sprintf("JOIN #%s", strings.Join(channels, ",#"))
	c.SendMessage([]byte(join))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgType, msg, err := c.conn.ReadMessage()
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
					// fmt.Println("USERNOTICE")
					parseUSERNOTICE(rawIRCMessage, c.C)
				case "PING":
					c.SendMessage([]byte("PONG :tmi.twitch.tv"))
				case "NOTICE":
					msg := parseNOTICE(rawIRCMessage)
					if msg.Err != nil {
						return err
					}
					if msg.SystemMsg == "Login authentication failed" {
						return ErrAuthFailed
					}
					c.writeToChannel(msg)
				}
			}
		}
	}
}
