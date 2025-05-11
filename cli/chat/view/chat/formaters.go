package chat

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/utils"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

func colorStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func GenerateIcon(userType string, colors config.Colors) string {
	switch userType {
	case "broadcaster":
		return colorStyle(colors.Icons.Broadcaster).Render(" [] ")
	case "mod":
		return " ✅"
	case "vip":
		return colorStyle(colors.Icons.Vip).Render(" [★] ")
	case "staff":
		return colorStyle(colors.Icons.Staff).Render(" [★] ")
	}
	return " "
}

// func formatMessageTimestamp(timestamp string, msg string) string {
// 	msgHeight := lipgloss.Height(msg)
// 	var newT string = timestamp
// 	for i := 1; i < msgHeight; i++ {
// 		newT += "\n" + strings.Repeat(" ", lipgloss.Width(timestamp))
// 	}
// 	return lipgloss.JoinHorizontal(1, newT, msg)
// }

func wrapText(s string, limit, padding int) string {
	var out strings.Builder
	// we keep track of the n char that were written
	var lineLen int

	// loop through every word in string
	for _, word := range strings.Fields(s) {
		// move to new row if length of the word + n char count is equal or bigger then limit
		if len(word)+lineLen >= limit {
			out.WriteString("\n")
			paddingStr := strings.Repeat(" ", padding)
			lineLen = padding
			if padding > 0 {
				out.WriteString(paddingStr)
			}
			// remainder := limit - lineLen
			// if len(word) >= remainder {
			// 	part := word[:remainder]
			// 	part2 := word[remainder+1:]
			// 	if padding > 0 {
			// 		word = part + "\n" + paddingStr + part2
			// 	} else {
			// 		word = part + "\n" + part2
			// 	}
			// 	lineLen = padding + len(part2)
			// }
		} else if lineLen > 0 {
			out.WriteString(" ")
			lineLen++
		}
		out.WriteString(word)
		lineLen += len(word)
	}

	return out.String()
}

func (m model) FormatChatMessage(message twitch.ChatMessage, width int) string {
	icon := GenerateIcon(message.Metadata.UserType, m.conf.Chat.Colors)
	if message.Metadata.Color == "" {
		message.Metadata.Color = string(rand.Intn(257))
	}

	var msgStr strings.Builder
	if icon != "" {
		msgStr.WriteString(icon + " ")
	}
	msgStr.WriteString(colorStyle(message.Metadata.Color).Render(message.Metadata.DisplayName) + ": ")
	msgStr.WriteString(message.Message)
	msgStyle := lipgloss.NewStyle()

	if !message.Metadata.IsFirstMessage {
		timestampMsg := fmt.Sprintf("[%s]", message.Metadata.Timestamp)
		timestamp := msgStyle.Faint(true).Render(timestampMsg)
		msg := wrapText(msgStr.String(), width-6, len(timestampMsg)+1)
		return fmt.Sprintf("%s %s", timestamp, strings.TrimSpace(msg))
	} else {
		firstMsgColor := m.conf.Chat.Colors.Messages.First
		box := NewBoxWithLabel(firstMsgColor)
		msg := wrapText(msgStr.String(), width-6, 0)
		return box.RenderBox(msgStyle.Foreground(lipgloss.Color(firstMsgColor)).Render(" First message "), msg)
	}
}

func (m model) FormatSubMessage(message twitch.SubNotice, width int) string {
	// if message.Metadata.Color == "" {
	// 	message.Metadata.Color = string(rand.Intn(257))
	// }
	msg := fmt.Sprintf(" ✯ %s", message.Metadata.SystemMsg)

	subColor := m.conf.Chat.Colors.Messages.Sub
	box := NewBoxWithLabel(subColor)
	msg = wordwrap.String(msg, width-50)
	msg = wrapText(msg, width-50, 0)
	color := lipgloss.Color(subColor)
	label := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf(" %s ", utils.Capitalize(message.SubPlan)))
	return box.RenderBox(label, msg)
}

func (m model) FormatRaidMessage(message twitch.RaidNotice, width int) string {
	icon := GenerateIcon(message.Metadata.UserType, m.conf.Chat.Colors)
	// if message.Metadata.Color == "" {
	// 	message.Metadata.Color = string(rand.Intn(257))
	// }
	msg := fmt.Sprintf(
		"%s %s: ✯ %s",
		icon,
		colorStyle(message.Metadata.Color).Render(message.Metadata.DisplayName),
		message.Metadata.SystemMsg,
	)
	raidColor := m.conf.Chat.Colors.Messages.Raid
	box := NewBoxWithLabel(raidColor)
	msg = wordwrap.String(msg, width-50)
	msg = wrapText(msg, width-50, 0)
	label := lipgloss.NewStyle().Foreground(lipgloss.Color(raidColor)).Render("Raid")
	return box.RenderBox(label, msg)
}

// func (m model) FormatGiftSubMessage(message SubGiftMessage, width int) string {
// 	box := NewBoxWithLabel(subColor)
// 	msg := fmt.Sprintf(
// 		"%s gifted a subscription to %s!",
// 		colorStyle(message.Color).Render(message.GiverName),
// 		message.ReceiverName,
// 	)
// 	msg = wordwrap.String(msg, width)
// 	if highlightSubs {
// 		return box.Render("Gift sub", msg)
// 	}
// 	return msg + "\n"
// }

// func (m model) FormatAnnouncementMessage(message AnnouncementMessage, width int) string {
// 	box := NewBoxWithLabel(announcementColor)
// 	msg := fmt.Sprintf(
// 		"%s: %s",
// 		colorStyle(message.Color).Render(message.DisplayName),
// 		message.Message,
// 	)
// 	msg = wordwrap.String(msg, width)
// 	return box.Render("Announcement", msg)
// }

// func (m model) FormatMysteryGiftSubMessage(message MysterySubGiftMessage, width int) string {
// 	box := NewBoxWithLabel(subColor)
// 	msg := fmt.Sprintf(
// 		"%s is giving %s subs to the channel!",
// 		colorStyle(message.Color).Render(message.GiverName),
// 		message.GiftAmount,
// 	)
// 	msg = wordwrap.String(msg, width)
// 	if highlightSubs {
// 		return box.Render("Gifting subs", msg)
// 	}
// 	return msg + "\n"
// }
