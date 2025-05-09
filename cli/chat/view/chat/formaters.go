package chat

import (
	"fmt"

	"github.com/Kostaaa1/twitch/internal/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

func colorStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func GenerateIcon(userType string) string {
	switch userType {
	case "broadcaster":
		return colorStyle("#d20f39").Render(" [] ")
	case "mod":
		return " ✅"
	case "vip":
		return colorStyle("#ea76cb").Render(" [★] ")
	case "staff":
		return colorStyle("#8839ef").Render(" [★] ")
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

func (m model) FormatChatMessage(message ChatMessage, width int) string {
	icon := GenerateIcon(message.Metadata.UserType)
	// if message.Metadata.Color == "" {
	// 	message.Metadata.Color = string(rand.Intn(257))
	// }
	msg := fmt.Sprintf(
		"%s %s: %s",
		icon,
		colorStyle(message.Metadata.Color).Render(message.Metadata.DisplayName),
		message.Message,
	)

	msg = wordwrap.String(msg, width-14)
	if !message.Metadata.IsFirstMessage {
		timestamp := lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("[%s]", message.Metadata.Timestamp))
		return fmt.Sprintf("%s%s", timestamp, msg)
	} else {
		firstMsgColor := m.conf.Chat.Colors.Messages.First
		box := NewBoxWithLabel(firstMsgColor)
		return box.RenderBox(lipgloss.NewStyle().Foreground(lipgloss.Color(firstMsgColor)).Render(" First message "), msg)
	}
}

func (m model) FormatSubMessage(message SubNotice, width int) string {
	// if message.Metadata.Color == "" {
	// 	message.Metadata.Color = string(rand.Intn(257))
	// }
	msg := fmt.Sprintf(" ✯ %s", message.Metadata.SystemMsg)

	subColor := m.conf.Chat.Colors.Messages.Sub
	box := NewBoxWithLabel(subColor)
	msg = wordwrap.String(msg, width-50)
	color := lipgloss.Color(subColor)
	label := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf(" %s ", utils.Capitalize(message.SubPlan)))
	return box.RenderBox(label, msg)
}

func (m model) FormatRaidMessage(message RaidNotice, width int) string {
	icon := GenerateIcon(message.Metadata.UserType)
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
