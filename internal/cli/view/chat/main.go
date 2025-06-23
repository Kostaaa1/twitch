package chat

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/cli/view/components"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/chat"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Chat struct {
	IsActive bool
	Channel  string
	Messages []string
	Room     chat.Room
}

type model struct {
	ws              *chat.WSClient
	conf            *config.Config
	viewport        viewport.Model
	labelBox        BoxWithLabel
	width           int
	height          int
	msgChan         chan interface{}
	chats           []Chat
	showHelpMenu    bool
	helperMenuWidth int
	notifyMsg       string
	footer          footer
}

type notifyMsg string

func ConnectWithRetry(ws *chat.WSClient, tw *twitch.Client, cfg *config.Config) error {
	err := ws.Connect()
	if err == nil {
		return nil
	}
	if errors.Is(err, chat.ErrAuthFailed) {
		if err := tw.RefetchAccesToken(); err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}
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

	t := textarea.New()
	t.CharLimit = 500
	t.Placeholder = "Send a message"
	t.Prompt = ""
	// t.Prompt = " ▶ "
	// t.Prompt = "┃ "
	t.FocusedStyle.CursorLine = lipgloss.NewStyle()
	t.ShowLineNumbers = false
	t.SetWidth(0)
	t.SetHeight(3)
	t.Cursor.Blink = true

	t.Focus()

	msgChan := make(chan interface{})

	ws, err := chat.DialWS(cfg.User.Login, cfg.Creds.AccessToken, cfg.Chat.OpenedChats)
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
		conf:            cfg,
		ws:              ws,
		chats:           chats,
		width:           0,
		height:          0,
		msgChan:         msgChan,
		labelBox:        NewBoxWithLabel(cfg.Chat.Colors.Primary),
		viewport:        vp,
		showHelpMenu:    false,
		helperMenuWidth: 32,
		footer:          NewFooter(t, 2),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
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

func (m *model) showNoActiveChatsMessage() {
	msg := "No active chats. Use '/add <channel_name>' to join channel."
	m.viewport.SetContent(lipgloss.NewStyle().Faint(true).Render(msg))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
	)

	m.footer.textarea, tiCmd = m.footer.textarea.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width - 2
		h := msg.Height - 8
		m.labelBox.SetWidth(w)
		m.viewport.Width = w
		m.width = w
		m.height = h
		m.viewport.Height = h - m.footer.height
		m.footer.textarea.SetWidth(w)
		m.viewport.Style = lipgloss.
			NewStyle().
			Width(m.viewport.Width).
			Height(m.viewport.Height)

		if len(m.chats) > 0 && m.chats[0].IsActive {
			m.updateChatViewport(&m.chats[0])
		} else if len(m.chats) == 0 {
			m.showNoActiveChatsMessage()
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.conf.Save()
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
			m.removeActiveChatAndDisconnect()
		case tea.KeyCtrlO:
		case tea.KeyTab:
			m.showHelpMenu = !m.showHelpMenu
			if m.showHelpMenu {
				newWidth := m.width - m.helperMenuWidth
				m.viewport.Width = newWidth
				// m.footer.textarea.Width = newWidth - m.footer.roomState.Len() - 4
			} else {
				m.viewport.Width = m.width
				// m.footer.textarea.Width = m.width - m.footer.roomState.Len() - 4
			}
		}

	case notifyMsg:
		return m, m.waitForMsg()

	case NewChannelMessage:
		switch chanMsg := msg.Data.(type) {
		case chat.Room:
			m.addRoomToChat(chanMsg)

		case chat.Message:
			chat := m.getChat(chanMsg.Metadata.RoomID)
			if chat != nil {
				m.appendMessage(chat, m.FormatMessage(chanMsg, m.width))
			}

		case chat.SubNotice:
			chat := m.getChat(chanMsg.Metadata.RoomID)
			if chat != nil {
				m.appendMessage(chat, m.FormatSubMessage(chanMsg, m.width))
			}

		case chat.Notice:
			if chanMsg.Err != nil {
				m.msgChan <- notifyMsg(chanMsg.SystemMsg)
				m.msgChan <- notifyMsg(chanMsg.Err.Error())
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
	if m.width == 0 {
		return ""
	}

	mainContentWidth := m.width
	if m.showHelpMenu {
		mainContentWidth -= m.helperMenuWidth
	}
	m.viewport.Width = mainContentWidth

	main := m.labelBox.SetWidth(m.viewport.Width).RenderBoxWithTabs(m.chats, m.viewport.View())

	mainArea := strings.Builder{}
	mainArea.WriteString(main)
	mainArea.WriteString(m.footer.Render(m))
	mainArea.WriteString(m.renderError())

	if !m.showHelpMenu {
		return mainArea.String()
	}

	helpMenu := components.RenderHelperMenu(m.helperMenuWidth, m.viewport.Height+m.footer.height+4)
	fullView := lipgloss.JoinHorizontal(
		lipgloss.Position(1),
		mainArea.String(),
		helpMenu,
	)

	return fullView
}

func (m *model) renderError() string {
	if m.notifyMsg != "" {
		return fmt.Sprintf("\n\n[ERROR] - %s", m.notifyMsg)
	}
	return ""
}

func (m *model) newMessage(newChat *Chat) chat.Message {
	newMessage := chat.Message{
		Message: m.footer.textarea.Value(),
		Metadata: chat.MessageMetadata{
			Metadata: chat.Metadata{
				Color:        newChat.Room.Metadata.Color,
				DisplayName:  newChat.Room.Metadata.DisplayName,
				IsMod:        newChat.Room.Metadata.IsMod,
				IsSubscriber: newChat.Room.Metadata.IsSubscriber,
				UserType:     newChat.Room.Metadata.UserType,
			},
			RoomID: newChat.Room.RoomID,
		},
	}
	return newMessage
}

func (m *model) sendMessage() {
	if m.footer.textarea.Value() == "" {
		return
	}

	input := m.footer.textarea.Value()

	if !strings.HasPrefix(input, "/") {
		chat := m.getActiveChat()
		if chat != nil {
			m.ws.FormatIRCMsgAndSend("PRIVMSG", chat.Channel, input)
			chat.Messages = append(chat.Messages, m.FormatMessage(m.newMessage(chat), m.width))
			m.updateChatViewport(chat)
		}
	} else {
		m.handleInputCommand(input)
	}

	m.footer.textarea.Reset()
}

func (m *model) handleInputCommand(cmd string) {
	parts := strings.Split(cmd, " ")
	if len(parts) > 2 || len(parts) < 2 {
		return
	}

	switch parts[0] {
	case "/add":
		if len(parts[1]) >= 25 {
			m.msgChan <- notifyMsg("Channel name is too long. Limit is 25 characters.")
			return
		}
		channelName := strings.TrimSpace(parts[1])
		m.addChat(channelName)
	case "/info":
		fmt.Println(parts[1])
	default:
		m.msgChan <- notifyMsg(fmt.Sprintf("invalid command: %s", cmd))
	}
}

func createNewChat(channel string, isActive bool) Chat {
	return Chat{
		IsActive: isActive,
		Messages: []string{
			lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("Welcome to %s channel", channel)),
		},
		Room:    chat.Room{},
		Channel: channel,
	}
}

func (m *model) addChat(channelName string) {
	newChat := createNewChat(channelName, true)
	m.chats = append(m.chats, newChat)
	m.ws.ConnectToChannel(newChat.Channel)
	m.updateChatViewport(&newChat)

	chats := []string{}
	for i := range m.chats {
		if m.chats[i].Channel != newChat.Channel {
			m.chats[i].IsActive = false
		}
		chats = append(chats, m.chats[i].Channel)
	}
	m.conf.Chat.OpenedChats = chats
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
		m.showNoActiveChatsMessage()
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
