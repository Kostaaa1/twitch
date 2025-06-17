package chat

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

type footer struct {
	roomState *roomState
	textarea  textarea.Model
	width     int
	height    int
}

func NewFooter(t textarea.Model, height int) footer {
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
		BorderForeground(lipgloss.Color(m.conf.Chat.Colors.Primary))

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			footer.roomState.render,
			footer.textarea.View(),
		),
	)

	// style := lipgloss.NewStyle().
	// 	Width(m.viewport.Width).
	// 	Height(footer.height).
	// 	Border(lipgloss.RoundedBorder()).
	// 	BorderForeground(lipgloss.Color(m.conf.Chat.Colors.Primary))

	// label := footer.roomState.render
	// return style.Render(label + "\n" + footer.textarea.View())
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

	style := lipgloss.NewStyle().Faint(true)

	switch {
	case chat.Room.FollowersOnly != "-1":
		m.footer.roomState.content = "[Followers-Only Chat]"
		m.footer.roomState.render = style.Render(m.footer.roomState.content)
	case chat.Room.IsSubsOnly:
		m.footer.roomState.content = "[Subscriber-Only Chat]"
		m.footer.roomState.render = style.Render(m.footer.roomState.content)
	case chat.Room.IsEmoteOnly:
		m.footer.roomState.content = "[Emote-Only Chat]"
		m.footer.roomState.render = style.Render(m.footer.roomState.content)
	}
}
