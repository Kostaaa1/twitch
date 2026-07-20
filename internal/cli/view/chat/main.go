package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/cli/view/components"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch/chat"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	errTimer           *time.Timer
	maxMessagesLimit   = 100
	maxOpenedChatLimit = 5
	faintStyle         = lipgloss.NewStyle().Faint(true)
)

type Chat struct {
	IsActive bool
	Channel  string
	Messages []string
	Room     chat.Room
}

type model struct {
	irc             *chat.TwitchIRC
	conf            *config.Config
	viewport        viewport.Model
	labelBox        BoxWithLabel
	width           int
	height          int
	chats           []Chat
	showHelpMenu    bool
	helperMenuWidth int
	notifyMsg       string
	footer          footer
}

type notifyMsg string

func DefaultScopes() []helix.Scope {
	return []helix.Scope{helix.ChatEdit, helix.ChatRead}
}

func Open(ctx context.Context, cfg *config.Config) error {
	vp := viewport.New(0, 0)
	vp.SetContent("")

	irc, err := chat.DialIRC(
		cfg.User.Login,
		cfg.OAuthCreds.UserToken.AccessToken,
		cfg.CommandLineChat.OpenedChats,
	)
	if err != nil {
		return err
	}

	go func() {
		if err := irc.Connect(
			ctx,
			cfg.OAuthCreds.UserToken.AccessToken,
			cfg.User.Login,
			cfg.CommandLineChat.OpenedChats,
		); err != nil {
			log.Fatal(err)
		}
	}()

	var chats []Chat
	for i, channel := range cfg.CommandLineChat.OpenedChats {
		chats = append(chats, createNewChat(channel, i == 0))
	}

	m := model{
		conf:            cfg,
		irc:             irc,
		chats:           chats,
		width:           0,
		height:          0,
		labelBox:        newBoxWithLabel(cfg.CommandLineChat.Colors.Primary),
		viewport:        vp,
		showHelpMenu:    false,
		helperMenuWidth: 32,
		footer:          newFooter(2),
	}

	if _, err := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		// tea.WithMouseAllMotion(),
	).Run(); err != nil {
		return nil
	}

	return nil
}

func (m model) Init() tea.Cmd {
	return m.waitForMsg()
}

type NewChannelMessage struct {
	Data interface{}
}

func (m model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		newMsg := <-m.irc.C
		switch newMsg.(type) {
		case notifyMsg:
			if errTimer != nil {
				errTimer.Stop()
			}
			errTimer = time.AfterFunc(time.Second*2, func() {
				m.irc.C <- newMsg
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
				m.irc.C <- notifyMsg(chanMsg.SystemMsg)
				m.irc.C <- notifyMsg(chanMsg.Err.Error())
				if err := m.irc.Close(); err != nil {
					fmt.Println(err)
				}
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
			m.irc.FormatIRCMsgAndSend("PRIVMSG", chat.Channel, input)
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
			m.irc.C <- notifyMsg("Channel name is too long. Limit is 25 characters.")
			return
		}
		channelName := strings.TrimSpace(parts[1])
		m.addChat(channelName)
	case "/info":
		fmt.Println(parts[1])
	default:
		m.irc.C <- notifyMsg(fmt.Sprintf("invalid command: %s", cmd))
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
	m.irc.ConnectToChannel(newChat.Channel)
	m.updateChatViewport(&newChat)

	chats := []string{}
	for i := range m.chats {
		if m.chats[i].Channel != newChat.Channel {
			m.chats[i].IsActive = false
		}
		chats = append(chats, m.chats[i].Channel)
	}
	m.conf.CommandLineChat.OpenedChats = chats
}

func (m *model) removeActiveChatAndDisconnect() {
	openedChats := m.conf.CommandLineChat.OpenedChats
	var chats []Chat
	newActiveId := -1

	for i, chat := range m.chats {
		if !chat.IsActive {
			chats = append(chats, chat)
		} else {
			openedChats = append(openedChats[:i], openedChats[i+1:]...)
			m.irc.LeaveChannel(chat.Channel)
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

	m.conf.CommandLineChat.OpenedChats = openedChats
	m.chats = chats
}

func (m *model) appendMessage(chat *Chat, message string) {
	if len(chat.Messages) > maxMessagesLimit {
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
