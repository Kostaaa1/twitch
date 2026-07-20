package chat

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

type footer struct {
	roomState *roomState
	textarea  textarea.Model
	width     int
	height    int
}

func newFooter(height int) footer {
	t := textarea.New()
	t.CharLimit = 500
	t.Placeholder = "Send a message"
	t.Prompt = ""
	t.FocusedStyle.CursorLine = lipgloss.NewStyle()
	t.ShowLineNumbers = false
	t.SetWidth(0)
	t.SetHeight(3)
	t.Cursor.Blink = true
	// t.Prompt = " ▶ "
	// t.Prompt = "┃ "
	t.Focus()

	return footer{
		roomState: new(roomState),
		textarea:  t,
		height:    height,
	}
}

func (footer footer) Render(m model) string {
	m.renderRoomState()

	style := lipgloss.NewStyle().
		Width(m.viewport.Width).
		Height(footer.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.conf.CommandLineChat.Colors.Primary))

	var str strings.Builder
	if footer.roomState.Len() > 0 {
		str.WriteString(footer.roomState.render)
		str.WriteString("\n")
	}
	str.WriteString(footer.textarea.View())

	return style.Render(str.String())
}

type roomState struct {
	content string
	render  string
}

func (s *roomState) Len() int {
	if s == nil {
		return 0
	}
	return len(s.content)
}

func (m model) renderRoomState() {
	chat := m.getActiveChat()
	if chat == nil {
		return
	}

	switch {
	case chat.Room.FollowersOnly != "-1":
		m.footer.roomState.content = "[Followers-Only Chat]"
		m.footer.roomState.render = faintStyle.Render(m.footer.roomState.content)
	case chat.Room.IsSubsOnly:
		m.footer.roomState.content = "[Subscriber-Only Chat]"
		m.footer.roomState.render = faintStyle.Render(m.footer.roomState.content)
	case chat.Room.IsEmoteOnly:
		m.footer.roomState.content = "[Emote-Only Chat]"
		m.footer.roomState.render = faintStyle.Render(m.footer.roomState.content)
	}
}
