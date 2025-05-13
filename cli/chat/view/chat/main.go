package chat

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/cli/chat/view/components"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/utils"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Chat struct {
	IsActive bool
	Channel  string
	Messages []string
	Room     twitch.Room
}

type model struct {
	twitch              *twitch.Client
	ws                  *twitch.WSClient
	conf                *config.Config
	viewport            viewport.Model
	labelBox            BoxWithLabel
	textinput           textinput.Model
	width               int
	height              int
	msgChan             chan interface{}
	chats               []Chat
	displayCommands     bool
	commandsWindowWidth int
	notifyMsg           string
}

type notifyMsg string

func ConnectWithRetry(ws *twitch.WSClient, tw *twitch.Client, cfg *config.Config) error {
	err := ws.Connect()
	if err == nil {
		return nil
	}

	if errors.Is(err, twitch.ErrAuthFailed) {
		if err := tw.RefetchAccesToken(); err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}
		// config.Save(tw.())
		if err := ws.Connect(); err != nil {
			return fmt.Errorf("retry connect failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("connect failed: %w", err)
}

func Open(tw *twitch.Client, cfg *config.Config) {
	vp := viewport.New(0, 0)
	vp.SetContent("")
	t := textinput.New()
	t.CharLimit = 500
	t.Placeholder = "Send a message"
	t.Prompt = " ▶ "
	t.Width = 50
	t.Focus()

	msgChan := make(chan interface{})

	ws, err := twitch.DialWS(cfg.User.Login, cfg.Creds.AccessToken, cfg.Chat.OpenedChats)
	if err != nil {
		log.Fatal(err)
	}
	ws.SetMessageChan(msgChan)

	go func() {
		if err := ConnectWithRetry(ws, tw, cfg); err != nil {
			log.Fatal(err)
		}
	}()

	var chats []Chat
	for i, channel := range cfg.Chat.OpenedChats {
		chats = append(chats, createNewChat(channel, i == 0))
	}

	m := model{
		conf:                cfg,
		twitch:              tw,
		ws:                  ws,
		chats:               chats,
		width:               0,
		height:              0,
		msgChan:             msgChan,
		labelBox:            NewBoxWithLabel(cfg.Chat.Colors.Primary),
		viewport:            vp,
		textinput:           t,
		displayCommands:     false,
		commandsWindowWidth: 32,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func createNewChat(channel string, isActive bool) Chat {
	return Chat{
		IsActive: isActive,
		Messages: []string{
			lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("Welcome to %s channel", channel)),
		},
		Room:    twitch.Room{},
		Channel: channel,
	}
}

func (m model) Init() tea.Cmd {
	return m.waitForMsg()
}

var errTimer *time.Timer

type NewChannelMessage struct {
	Data interface{}
}

func (m model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		newMsg := <-m.msgChan
		switch newMsg.(type) {
		case notifyMsg:
			if errTimer != nil {
				errTimer.Stop()
			}
			errTimer = time.AfterFunc(time.Second*2, func() {
				m.msgChan <- newMsg
			})
			return newMsg
		default:
			return NewChannelMessage{Data: newMsg}
		}
	}
}

func (m *model) showNoChatMessage() {
	msg := "No active chats. Use '/add <channel_name>' to join channel."
	m.viewport.SetContent(lipgloss.NewStyle().Faint(true).Render(msg))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
	)
	m.textinput, tiCmd = m.textinput.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width - 2
		h := msg.Height - 8
		m.labelBox.SetWidth(w)
		m.viewport.Width = w
		m.viewport.Height = h
		m.width = w
		m.height = h
		m.viewport.Style = lipgloss.
			NewStyle().
			Width(m.viewport.Width).
			Height(m.viewport.Height)

		if len(m.chats) > 0 && m.chats[0].IsActive {
			m.updateChatViewport(&m.chats[0])
		} else if len(m.chats) == 0 {
			m.showNoChatMessage()
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			config.Save(m.conf)
			return m, tea.Quit
		case tea.KeyEnter:
			m.sendMessage()
		case tea.KeyCtrlRight:
			m.nextTab()
		case tea.KeyCtrlLeft:
			m.prevTab()
		case tea.KeyCtrlShiftRight:
			m.moveTabForward()
		case tea.KeyCtrlShiftLeft:
			m.moveTabBack()
		case tea.KeyCtrlW:
			// if len(m.chats) > 1 {
			m.removeActiveChatAndDisconnect()
			// }
		case tea.KeyCtrlO:
			go func() { // check if safe
				chat := m.getActiveChat()
				if chat != nil {
					// master, err := m.twitch.GetStreamMasterPlaylist(chat.Channel)
					// if err != nil {
					// 	m.msgChan <- notifyMsg(err.Error())
					// 	return
					// }
					// list, err := master.GetVariantPlaylistByQuality("best")
					// if err != nil {
					// 	m.msgChan <- notifyMsg(err.Error())
					// 	return
					// }
					// cmd := exec.Command("vlc", list.URL)
					// if err := cmd.Run(); err != nil {
					// 	m.msgChan <- notifyMsg(err.Error())
					// 	return
					// }
					// cmd.Wait()
					m.msgChan <- notifyMsg("VLC closed")
				}
			}()
		case tea.KeyTab:
			m.displayCommands = !m.displayCommands
			if m.displayCommands {
				m.viewport.Width = m.width - m.commandsWindowWidth
			} else {
				m.viewport.Width = m.width
			}
		}

	case notifyMsg:
		return m, m.waitForMsg()

	case NewChannelMessage:
		switch chanMsg := msg.Data.(type) {
		case twitch.Room:
			m.addRoomToChat(chanMsg)

		case twitch.ChatMessage:
			chat := m.getChat(chanMsg.Metadata.RoomID)
			if chat != nil {
				m.appendMessage(chat, m.FormatChatMessage(chanMsg, m.width))
			}

		case twitch.SubNotice:
			chat := m.getChat(chanMsg.Metadata.RoomID)
			if chat != nil {
				m.appendMessage(chat, m.FormatSubMessage(chanMsg, m.width))
			}

		case twitch.Notice:
			if chanMsg.Err != nil {
				m.msgChan <- notifyMsg(chanMsg.SystemMsg)
				m.msgChan <- notifyMsg(chanMsg.Err.Error())
			}

			if chanMsg.Err != nil {
				m.ws.Conn.Close()
			}

			chat := m.getChat(chanMsg.DisplayName)
			if chat != nil {
				m.appendMessage(chat, chanMsg.SystemMsg)
			}
		}
		return m, m.waitForMsg()
	}
	return m, tea.Batch(tiCmd)
}

func (m model) View() string {
	var b strings.Builder
	main := m.labelBox.SetWidth(m.viewport.Width).RenderBoxWithTabs(m.chats, m.viewport.View())
	if !m.displayCommands {
		b.WriteString(main)
	} else {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Position(0.5), main, components.RenderCommands(m.commandsWindowWidth, m.height)))
	}
	b.WriteString("\n" + lipgloss.JoinHorizontal(lipgloss.Position(0), m.renderRoomState(), m.textinput.View()))
	b.WriteString(m.renderError())
	return b.String()
}

func (m *model) newChatMessage(chat *Chat) twitch.ChatMessage {
	newMessage := twitch.ChatMessage{
		Message: m.textinput.Value(),
		Metadata: twitch.ChatMessageMetadata{
			Metadata: twitch.Metadata{
				Color:        chat.Room.Metadata.Color,
				DisplayName:  chat.Room.Metadata.DisplayName,
				IsMod:        chat.Room.Metadata.IsMod,
				IsSubscriber: chat.Room.Metadata.IsSubscriber,
				UserType:     chat.Room.Metadata.UserType,
			},
			RoomID:    chat.Room.RoomID,
			Timestamp: utils.GetCurrentTimeFormatted(),
		},
	}
	return newMessage
}

func (m *model) renderError() string {
	var b strings.Builder
	if m.notifyMsg != "" {
		b.WriteString(fmt.Sprintf("\n\n[ERROR] - %s", m.notifyMsg))
	} else {
		b.WriteString("")
	}
	return b.String()
}

func (m *model) sendMessage() {
	if m.textinput.Value() == "" {
		return
	}
	input := m.textinput.Value()
	if !strings.HasPrefix(input, "/") {
		chat := m.getActiveChat()
		if chat != nil {
			newMessage := m.newChatMessage(chat)
			m.ws.FormatIRCMsgAndSend("PRIVMSG", chat.Channel, input)
			chat.Messages = append(chat.Messages, m.FormatChatMessage(newMessage, m.width))
			m.updateChatViewport(chat)
		}
	} else {
		m.handleInputCommand(input)
	}
	m.textinput.Reset()
}

func (m *model) handleInputCommand(cmd string) {
	parts := strings.Split(cmd, " ")
	if len(parts) > 2 || len(parts) < 2 {
		return
	}

	switch parts[0] {
	case "/add":
		m.addChat(parts[1])
	case "/info":
		fmt.Println(parts[1])
	default:
		m.msgChan <- notifyMsg(fmt.Sprintf("invalid command: %s", cmd))
	}
}

func (m *model) addChat(channelName string) {
	newChat := createNewChat(channelName, true)
	m.chats = append(m.chats, newChat)
	m.ws.ConnectToChannel(newChat.Channel)
	m.updateChatViewport(&newChat)

	// For config
	chats := []string{}
	for i := range m.chats {
		if m.chats[i].Channel != newChat.Channel {
			m.chats[i].IsActive = false
		}
		chats = append(chats, m.chats[i].Channel)
	}
	m.conf.Chat.OpenedChats = chats
}

func (m *model) addRoomToChat(chanMsg twitch.Room) {
	for i := range m.chats {
		c := &(m.chats)[i]
		if c.Channel == chanMsg.Metadata.Channel {
			c.Room = chanMsg
			break
		}
	}
}

func (m *model) removeActiveChatAndDisconnect() {
	openedChats := m.conf.Chat.OpenedChats
	var chats []Chat
	newActiveId := -1

	for i, chat := range m.chats {
		if !chat.IsActive {
			chats = append(chats, chat)
		} else {
			openedChats = append(openedChats[:i], openedChats[i+1:]...)
			m.ws.LeaveChannel(chat.Channel)
			newActiveId = i
			if i == len(m.chats)-1 {
				newActiveId--
			}
		}
	}

	if newActiveId > -1 {
		chats[newActiveId].IsActive = true
		chat := chats[newActiveId]
		m.updateChatViewport(&chat)
	} else {
		m.showNoChatMessage()
	}

	m.conf.Chat.OpenedChats = openedChats
	m.chats = chats
}

func (m *model) appendMessage(chat *Chat, message string) {
	if len(chat.Messages) > 100 {
		chat.Messages = chat.Messages[1:]
	}
	chat.Messages = append(chat.Messages, message)
	if chat.IsActive {
		m.updateChatViewport(chat)
	}
}

func (m *model) updateChatViewport(chat *Chat) {
	m.viewport.SetContent(strings.Join(chat.Messages, "\n"))
	m.viewport.GotoBottom()
}

func (m *model) moveTabForward() {
	openedChats := make([]string, len(m.chats))
	for i := len(m.chats) - 1; i >= 0; i-- {
		if i > 0 && m.chats[i-1].IsActive {
			m.chats[i], m.chats[i-1] = m.chats[i-1], m.chats[i]
		}
		openedChats[i] = m.chats[i].Channel
	}
	m.conf.Chat.OpenedChats = openedChats
}

func (m *model) moveTabBack() {
	openedChats := make([]string, len(m.chats))
	for i := range m.chats {
		if i < len(m.chats)-1 && m.chats[i+1].IsActive {
			m.chats[i], m.chats[i+1] = m.chats[i+1], m.chats[i]
		}
		openedChats[i] = m.chats[i].Channel
	}
	m.conf.Chat.OpenedChats = openedChats
}

func (m *model) nextTab() {
	var activeIndex int
	for i, chat := range m.chats {
		if chat.IsActive {
			activeIndex = i
			break
		}
	}
	(m.chats)[activeIndex].IsActive = false
	nextIndex := (activeIndex + 1) % len(m.chats)
	(m.chats)[nextIndex].IsActive = true
	m.updateChatViewport(&(m.chats)[nextIndex])
}

func (m *model) prevTab() {
	var activeIndex int
	for i, c := range m.chats {
		if c.IsActive {
			activeIndex = i
			break
		}
	}
	(m.chats)[activeIndex].IsActive = false
	prevIndex := (activeIndex - 1 + len(m.chats)) % len(m.chats)
	(m.chats)[prevIndex].IsActive = true
	m.updateChatViewport(&(m.chats)[prevIndex])
}

func (m model) getActiveChat() *Chat {
	for i := range m.chats {
		if (m.chats)[i].IsActive {
			return &(m.chats[i])
		}
	}
	return nil
}

func (m model) getChat(roomID string) *Chat {
	for i := range m.chats {
		if (m.chats)[i].Room.RoomID == roomID || (m.chats)[i].Channel == roomID {
			return &(m.chats[i])
		}
	}
	return nil
}

func (m model) renderRoomState() string {
	chat := m.getActiveChat()
	if chat == nil {
		return ""
	}

	style := lipgloss.NewStyle().Faint(true)
	switch {
	case chat.Room.FollowersOnly != "-1":
		return style.Render("[Followers-Only Chat]")
	case chat.Room.IsSubsOnly:
		return style.Render("[Subscriber-Only Chat]")
	case chat.Room.IsEmoteOnly:
		return style.Render("[Emote-Only Chat]")
	default:
		return ""
	}
}
