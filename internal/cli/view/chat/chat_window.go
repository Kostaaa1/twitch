package chat

import "github.com/Kostaaa1/twitch/pkg/twitch/chat"

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

func (m *model) addRoomToChat(chanMsg chat.Room) {
	for i := range m.chats {
		c := &(m.chats)[i]
		if c.Channel == chanMsg.Metadata.Channel {
			c.Room = chanMsg
			break
		}
	}
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
