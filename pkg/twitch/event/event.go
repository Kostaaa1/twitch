package event

type Type string

var (
	ChannelFollow           Type = "channel.follow"
	ChannelAdBreakBegin     Type = "channel.ad_break.begin"
	ChannelUpdate           Type = "channel.update"
	ChannelBan              Type = "channel.ban"
	ChannelUnban            Type = "channel.unban"
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

func (sub *EventSub) StreamOnlineEvent(channelName string) (Event, error) {
	user, err := sub.tw.User(nil, &channelName)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Version:   1,
		Type:      StreamOnline,
		Condition: map[string]interface{}{"broadcaster_user_id": user.ID},
	}, nil
}
