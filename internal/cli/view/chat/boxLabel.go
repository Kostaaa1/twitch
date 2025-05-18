package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type BoxWithLabel struct {
	BoxStyle   lipgloss.Style
	LabelStyle lipgloss.Style
	color      lipgloss.Color
	width      int
}

func NewBoxWithLabel(color string) BoxWithLabel {
	c := lipgloss.Color(color)
	return BoxWithLabel{
		BoxStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c).
			Padding(0),
		LabelStyle: lipgloss.
			NewStyle().
			Padding(0),
		color: c,
	}
}

func (b *BoxWithLabel) SetWidth(width int) *BoxWithLabel {
	b.width = width
	return b
}

func (b *BoxWithLabel) renderTab(chat Chat, id int) string {
	border := lipgloss.Border{
		Top:         "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "│",
		BottomRight: "╰",
		Bottom:      "─",
	}

	if chat.IsActive {
		border.Bottom = " "
		if id == 0 {
			border.BottomLeft = "│"
		} else {
			border.BottomRight = "╰"
			border.BottomLeft = "╯"
		}
	} else {
		if id == 0 {
			border.BottomLeft = "├"
			border.BottomRight = "┴"
		} else {
			border.BottomRight = "┴"
			border.BottomLeft = "┴"
		}
	}

	l := b.LabelStyle.Border(border).
		BorderForeground(b.color).
		Bold(true).
		Italic(true).
		Padding(0)

	if chat.IsActive {
		l = l.Foreground(b.color)
	}
	return l.Render(fmt.Sprintf(" %s ", chat.Channel))
}

func (b *BoxWithLabel) RenderBoxWithTabs(chats []Chat, content string) string {
	var (
		topBorderStyler func(strs ...string) string = lipgloss.NewStyle().
				Foreground(b.BoxStyle.GetBorderTopForeground()).
				Render
		border      lipgloss.Border = b.BoxStyle.GetBorderStyle()
		borderTop   string          = topBorderStyler(border.Top)
		borderLeft  string          = topBorderStyler(border.TopLeft)
		borderRight string          = topBorderStyler(border.TopRight)
	)

	width := lipgloss.Width(content)
	if b.width != 0 {
		width = b.width
	}
	borderWidth := b.BoxStyle.GetHorizontalBorderSize()

	var stack []string
	for i := range chats {
		stack = append(stack, b.renderTab(chats[i], i))
	}
	labels := lipgloss.JoinHorizontal(lipgloss.Position(0), stack...)

	cellsShort := max(1, width+borderWidth-lipgloss.Width(borderLeft+borderRight+labels))
	gap := strings.Repeat(border.Top, cellsShort)

	var top string
	if len(stack) == 0 {
		top = "\n" + "\n" + borderLeft + topBorderStyler(gap) + borderRight
	} else {
		top = labels + topBorderStyler(gap) + borderTop + borderRight
	}

	bottom := b.BoxStyle.
		BorderTop(false).
		Width(width).
		Render(content)

	return top + "\n" + bottom + "\n"
}

func (b *BoxWithLabel) RenderBox(label, content string) string {
	var (
		topBorderStyler func(strs ...string) string = lipgloss.NewStyle().
				Foreground(b.BoxStyle.GetBorderTopForeground()).
				Render
		border        lipgloss.Border = b.BoxStyle.GetBorderStyle()
		topLeft       string          = topBorderStyler(border.TopLeft)
		topRight      string          = topBorderStyler(border.TopRight)
		renderedLabel string          = b.LabelStyle.
				Bold(true).
				Italic(true).
				Padding(0).
				Render(label)
	)

	width := lipgloss.Width(content)
	if b.width != 0 {
		width = b.width
	}
	borderWidth := b.BoxStyle.GetHorizontalBorderSize()
	cellsShort := max(0, width+borderWidth-lipgloss.Width(topLeft+topRight+renderedLabel))
	gap := strings.Repeat(border.Top, cellsShort)
	top := topBorderStyler(border.TopLeft) + renderedLabel + topBorderStyler(gap) + topRight

	if width < lipgloss.Width(top) {
		content = content + strings.Repeat(" ", lipgloss.Width(top)-width-2)
		width = lipgloss.Width(top) - 2
	}

	bottom := b.BoxStyle.
		BorderTop(false).
		Width(width).
		Render(content)

	return top + "\n" + bottom
}
