package chat

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	Conn *websocket.Conn
}

func createWSClient() (*WebSocketClient, error) {
	socketURL := "ws://irc-ws.chat.twitch.tv:80"
	conn, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		return nil, err
	}
	return &WebSocketClient{Conn: conn}, nil
}

func (c *WebSocketClient) SendMessage(msg []byte) error {
	return c.Conn.WriteMessage(websocket.TextMessage, msg)
}

func (c *WebSocketClient) FormatIRCMsgAndSend(tag, channel, msg string) error {
	formatted := fmt.Sprintf("%s #%s :%s", tag, channel, msg)
	return c.SendMessage([]byte(formatted))
}

func (c *WebSocketClient) LeaveChannel(channel string) {
	part := fmt.Sprintf("PART #%s", channel)
	c.SendMessage([]byte(part))
}

func (c *WebSocketClient) ConnectToChannel(channel string) {
	join := fmt.Sprintf("JOIN #%s", channel)
	c.SendMessage([]byte(join))
}

func (c *WebSocketClient) ConnectToIRC(accessToken, username string, msgChan chan interface{}, channels []string) error {
	c.SendMessage([]byte("CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands"))

	pass := fmt.Sprintf("PASS oauth:%s", accessToken)
	c.SendMessage([]byte(pass))

	nick := fmt.Sprintf("NICK %s", username)
	c.SendMessage([]byte(nick))

	join := fmt.Sprintf("JOIN #%s", strings.Join(channels, ",#"))
	c.SendMessage([]byte(join))

	pattern := `\b(PING|PRIVMSG|ROOMSTATE|USERNOTICE|USERSTATE|NOTICE|GLOBALUSERSTATE|CLEARMSG|CLEARCHAT)\b`
	re := regexp.MustCompile(pattern)

	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			return err
		}

		if msgType == websocket.TextMessage {
			rawIRCMessage := strings.TrimSpace(string(msg))

			msgChan <- rawIRCMessage
			tags := re.FindStringSubmatch(rawIRCMessage)
			if len(tags) > 1 {
				tag := tags[1]
				switch tag {
				case "USERSTATE":
					m := parseROOMSTATE(rawIRCMessage)
					msgChan <- m
				case "PRIVMSG":
					parsed := parsePRIVMSG(rawIRCMessage)
					msgChan <- parsed
				case "USERNOTICE":
					parseUSERNOTICE(rawIRCMessage, msgChan)
				case "PING":
					c.SendMessage([]byte("PONG :tmi.twitch.tv"))
				case "NOTICE":
					parsed := parseNOTICE(rawIRCMessage)
					msgChan <- parsed
				}
			}
		}
	}
}
