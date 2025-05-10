package twitch

type Metadata struct {
	Color        string
	DisplayName  string
	IsMod        bool
	IsSubscriber bool
	UserType     string
}

type ChatMessageMetadata struct {
	Metadata
	RoomID         string
	IsFirstMessage bool
	Timestamp      string
}

type ChatMessage struct {
	Metadata ChatMessageMetadata
	Message  string
}

type RoomMetadata struct {
	Metadata
	Channel string
}

type Room struct {
	Metadata      RoomMetadata
	RoomID        string
	IsEmoteOnly   bool
	FollowersOnly string
	IsSubsOnly    bool
}

type NoticeMetadata struct {
	Metadata
	MsgID     string
	RoomID    string
	SystemMsg string
	Timestamp string
	UserID    string
}

type RaidNotice struct {
	Metadata         NoticeMetadata
	ParamDisplayName string
	ParamLogin       string
	ViewerCount      int
}

type SubGiftNotice struct {
	Metadata             NoticeMetadata
	Months               int
	RecipientDisplayName string
	RecipientID          string
	RecipientName        string
	SubPlan              string
}

type SubNotice struct {
	Metadata  NoticeMetadata
	Months    int
	SubPlan   string
	WasGifted bool
}

type Notice struct {
	MsgID       string
	DisplayName string
	SystemMsg   string
	Err         error
}
