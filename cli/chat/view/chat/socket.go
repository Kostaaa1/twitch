package chat

// type WebSocketClient struct {
// 	Conn        *websocket.Conn
// 	Username    string
// 	Channels    []string
// 	AccessToken string
// 	ch          chan interface{}
// }

// func DialIRC(name, token string, channels []string) (*WebSocketClient, error) {
// 	socketURL := "ws://irc-ws.chat.twitch.tv:80"
// 	conn, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &WebSocketClient{
// 		Conn:        conn,
// 		Username:    name,
// 		AccessToken: token,
// 		Channels:    channels,
// 	}, nil
// }

// func (c *WebSocketClient) SetMessageChan(ch chan interface{}) {
// 	c.ch = ch
// }

// func (c *WebSocketClient) SendMessage(msg []byte) error {
// 	return c.Conn.WriteMessage(websocket.TextMessage, msg)
// }

// func (c *WebSocketClient) FormatIRCMsgAndSend(tag, channel, msg string) error {
// 	formatted := fmt.Sprintf("%s #%s :%s", tag, channel, msg)
// 	return c.SendMessage([]byte(formatted))
// }

// func (c *WebSocketClient) LeaveChannel(channel string) {
// 	part := fmt.Sprintf("PART #%s", channel)
// 	c.SendMessage([]byte(part))
// }

// func (c *WebSocketClient) ConnectToChannel(channel string) {
// 	join := fmt.Sprintf("JOIN #%s", channel)
// 	c.SendMessage([]byte(join))
// }

// func (c *WebSocketClient) Connect() error {
// 	c.SendMessage([]byte("CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands"))

// 	pass := fmt.Sprintf("PASS oauth:%s", c.AccessToken)
// 	c.SendMessage([]byte(pass))

// 	nick := fmt.Sprintf("NICK %s", c.Username)
// 	c.SendMessage([]byte(nick))

// 	join := fmt.Sprintf("JOIN #%s", strings.Join(c.Channels, ",#"))
// 	c.SendMessage([]byte(join))

// 	pattern := `\b(PING|PRIVMSG|ROOMSTATE|USERNOTICE|USERSTATE|NOTICE|GLOBALUSERSTATE|CLEARMSG|CLEARCHAT)\b`
// 	re := regexp.MustCompile(pattern)

// 	for {
// 		msgType, msg, err := c.Conn.ReadMessage()
// 		if err != nil {
// 			log.Printf("Error reading WebSocket message: %v", err)
// 			return err
// 		}

// 		if msgType == websocket.TextMessage {
// 			rawIRCMessage := strings.TrimSpace(string(msg))
// 			fmt.Println("RAW IRC MESSAGE:", rawIRCMessage)

// 			c.ch <- rawIRCMessage
// 			tags := re.FindStringSubmatch(rawIRCMessage)
// 			if len(tags) > 1 {
// 				tag := tags[1]
// 				switch tag {
// 				case "USERSTATE":
// 					m := parseROOMSTATE(rawIRCMessage)
// 					c.ch <- m
// 				case "PRIVMSG":
// 					parsed := parsePRIVMSG(rawIRCMessage)
// 					c.ch <- parsed
// 				case "USERNOTICE":
// 					parseUSERNOTICE(rawIRCMessage, c.ch)
// 				case "PING":
// 					c.SendMessage([]byte("PONG :tmi.twitch.tv"))
// 				case "NOTICE":
// 					parsed := parseNOTICE(rawIRCMessage)
// 					if parsed.SystemMsg == "Login authentication failed" {
// 						return err
// 					}
// 					c.ch <- parsed
// 				}
// 			}
// 		}
// 	}
// }
