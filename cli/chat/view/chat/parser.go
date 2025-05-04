package chat

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Kostaaa1/twitch/internal/utils"
)

func parseROOMSTATE(rawMsg string) Room {
	var parts []string
	metadata := strings.Split(rawMsg, "@")
	var room = Room{
		Metadata: RoomMetadata{},
	}
	if len(metadata) < 3 {
		return room
	}

	userParts := strings.Split(metadata[1], " :")
	room.Metadata.Channel = strings.TrimSpace(strings.Split(userParts[1], "#")[1])
	userMD := userParts[0]
	roomMD := strings.Split(metadata[2], " :")[0]

	parts = append(parts, strings.Split(userMD, ";")...)
	parts = append(parts, strings.Split(roomMD, ";")...)
	parseMetadata(&room.Metadata, parts)

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) > 1 {
			key := kv[0]
			value := kv[1]
			switch key {
			case "room-id":
				room.RoomID = value
			case "emote-only":
				room.IsEmoteOnly = value == "1"
			case "followers-only":
				room.FollowersOnly = value
			case "subs-only":
				room.IsSubsOnly = value == "1"
			}
		}
	}
	return room
}

func parsePRIVMSG(msg string) ChatMessage {
	emojiRx := regexp.MustCompile(`[^\p{L}\p{N}\p{Zs}:/?&=.-@]+`)
	parts := strings.SplitN(msg, " :", 2)
	extracted := strings.TrimSpace(strings.Split(parts[1], " :")[1])
	message := ChatMessage{
		Message:  emojiRx.ReplaceAllString(extracted, ""),
		Metadata: ChatMessageMetadata{},
	}

	mdParts := strings.Split(parts[0], ";")
	unusedPairs := parseMetadata(&message.Metadata, mdParts)
	for _, pair := range unusedPairs {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			key := kv[0]
			value := kv[1]
			switch key {
			case "first-msg":
				message.Metadata.IsFirstMessage = value == "1"
			}
		}
	}
	return message
}

func parseSubPlan(plan string) string {
	if plan == "1000" {
		return "Tier 1"
	}
	if plan == "2000" {
		return "Tier 2"
	}
	if plan == "3000" {
		return "Tier 3"
	}
	return "Prime"
}

func parseSubGiftMessage(pairs []string, notice *SubGiftNotice) {
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			key := kv[0]
			value := kv[1]
			switch key {
			case "msg-param-months":
				n, _ := strconv.Atoi(value)
				notice.Months = n
			case "msg-param-recipient-display-name":
				notice.RecipientDisplayName = value
			case "msg-param-recipient-id":
				notice.RecipientID = value
			case "msg-param-recipient-name":
				notice.RecipientName = value
			case "msg-param-sub-plan":
				notice.SubPlan = parseSubPlan(value)
			}
		}
	}
}

func parseRaidNotice(pairs []string, raidNotice *RaidNotice) {
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			key := kv[0]
			value := kv[1]
			switch key {
			case "msg-param-viewerCount":
				n, _ := strconv.Atoi(value)
				raidNotice.ViewerCount = n
			}
		}
	}
}

func parseSubNotice(pairs []string, notice *SubNotice) {
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			key := kv[0]
			value := kv[1]
			switch key {
			case "msg-param-cumulative-months":
				n, _ := strconv.Atoi(value)
				notice.Months = n
			case "msg-param-sub-plan":
				notice.SubPlan = parseSubPlan(value)
			case "msg-param-was-gifted":
				notice.WasGifted = value == "true"
			}
		}
	}
}

func parseMetadata(metadata interface{}, pairs []string) []string {
	var notUsedValues []string
	parseBaseMetadata := func(m *Metadata, key, value, pair string) {
		switch key {
		case "color":
			m.Color = value
		case "display-name":
			m.DisplayName = value
		case "mod":
			m.IsMod = value == "1"
		case "subscriber":
			m.IsSubscriber = value == "1"
		case "user-type":
			m.UserType = value
		default:
			if value != "" {
				notUsedValues = append(notUsedValues, pair)
			}
		}
	}
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			key := kv[0]
			value := kv[1]
			switch m := metadata.(type) {
			case *RoomMetadata:
				parseBaseMetadata(&m.Metadata, key, value, pair)
			case *NoticeMetadata:
				parseBaseMetadata(&m.Metadata, key, value, pair)
				switch key {
				case "msg-id":
					m.MsgID = value
				case "room-id":
					m.RoomID = value
				case "system-msg":
					m.SystemMsg = strings.Join(strings.Split(value, `\s`), " ")
				case "tmi-sent-ts":
					m.Timestamp = utils.ParseTimestamp(value)
				case "user-id":
					m.UserID = value
				}
			case *ChatMessageMetadata:
				parseBaseMetadata(&m.Metadata, key, value, pair)
				switch key {
				case "room-id":
					m.RoomID = value
				case "tmi-sent-ts":
					m.Timestamp = utils.ParseTimestamp(value)
				}
			}
		}
	}
	return notUsedValues
}

func parseUSERNOTICE(rawMsg string, msgChan chan interface{}) {
	parts := strings.SplitN(rawMsg[1:], " :", 2)
	pairs := strings.Split(parts[0], ";")
	var metadata NoticeMetadata
	notUsedPairs := parseMetadata(&metadata, pairs)

	switch metadata.MsgID {
	case "sub":
		var resubNotice = SubNotice{
			Metadata: metadata,
		}
		parseSubNotice(notUsedPairs, &resubNotice)
		msgChan <- resubNotice
	case "resub":
		var resubNotice = SubNotice{
			Metadata: metadata,
		}
		parseSubNotice(notUsedPairs, &resubNotice)
		msgChan <- resubNotice
	case "raid":
		var raidNotice = RaidNotice{
			Metadata: metadata,
		}
		parseRaidNotice(notUsedPairs, &raidNotice)
		msgChan <- raidNotice
	case "subgift":
		var notice = SubGiftNotice{
			Metadata: metadata,
		}
		parseSubGiftMessage(notUsedPairs, &notice)
		msgChan <- notice
	}
}

func isNoticeValid(notice Notice) {
	if notice.MsgID == "Login authentication failed" {
		return
	}
	if notice.MsgID == "Improperly formatted auth" {
		return
	}
}

// Did not handle the error cases properly, where login failed, unauthenticated (due to not having permissions for IRC), maybe more...
func parseNOTICE(rawMsg string) Notice {
	var notice Notice
	fmt.Println("Raw irc message", rawMsg)

	parts := strings.Split(rawMsg[1:], " :")

	// if len(parts) < 3 {
	// 	notice.Err = fmt.Errorf("parsing error: %s", parts[1])
	// 	return notice
	// }

	if len(parts) == 2 {
		notice.SystemMsg = parts[1]
	}

	if len(parts) >= 3 {
		notice.MsgID = strings.Split(parts[0], "=")[1]
		notice.DisplayName = strings.Split(parts[1], "#")[1]
	}

	return notice
}
