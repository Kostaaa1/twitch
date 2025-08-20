package event

import (
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
)

type Type string

var (
	ChannelFollow           Type = "channel.follow"
	ChannelSubscribe        Type = "channel.subscribe"
	ChannelAdBreakBegin     Type = "channel.ad_break.begin"
	ChannelUpdate           Type = "channel.update"
	ChannelBan              Type = "channel.ban"
	ChannelUnban            Type = "channel.unban"
	ChannelPollBegin        Type = "channel.poll.begin"
	ChannelPollProgress     Type = "channel.poll.progress"
	ChannelPollEnd          Type = "channel.poll.end"
	StreamOnline            Type = "stream.online"
	StreamOffline           Type = "stream.offline"
	UserAuthorizationGrant  Type = "user.authorization.grant"
	UserAuthorizationRevoke Type = "user.authorization.revoke"
	UserUpdate              Type = "user.update"
	WhisperReceived         Type = "user.whisper.message"
)

type Event struct {
	Version   int32                  `json:"version"`
	Type      Type                   `json:"type"`
	Condition map[string]interface{} `json:"condition"`
}

func StreamOnlineEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      StreamOnline,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func StreamOfflineEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      StreamOffline,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func ChannelAdBreakBeginEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      ChannelAdBreakBegin,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func ChannelFollowEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      ChannelFollow,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func ChannelUpdateEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      ChannelUpdate,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func ChannelSubscribeEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      ChannelSubscribe,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func ChannelPollBeginEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      ChannelPollBegin,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func ChannelPollProgressEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      ChannelPollProgress,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

func ChannelPollEndEvent(userID string) Event {
	return Event{
		Version:   1,
		Type:      ChannelPollEnd,
		Condition: map[string]interface{}{"broadcaster_user_id": userID},
	}
}

// uses userID to
func FromUnits(units []downloader.Unit) ([]Event, error) {
	var events []Event
	for _, unit := range units {
		if unit.Error != nil {
			return nil, unit.Error
		}
		if unit.Type == downloader.TypeLivestream {
			// events = append(events, StreamOnlineEvent(unit.UserID))
		}
	}
	return events, nil
}
