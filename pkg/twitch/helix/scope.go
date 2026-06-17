package helix

type Scope string

const (
	// View analytics data for the Twitch Extensions owned by the authenticated account.
	// API
	// Get Extension Analytics
	AnalyticsReadExtensions Scope = "analytics:read:extensions"

	// View analytics data for the games owned by the authenticated account.
	// API
	// Get Game Analytics
	AnalyticsReadGames Scope = "analytics:read:games"

	// View Bits-related products and redemptions for a channel.
	// API
	// Get Bits Leaderboard
	// Get Custom Power-up
	// EventSub
	// Channel Bits Use
	// Channel Cheer
	// Channel Custom Power-ups Redemption Add
	BitsRead Scope = "bits:read"

	// Joins your channel's chatroom as a bot user, and perform chat-related actions as that user.
	// API
	// Send Chat Message
	// EventSub
	// Channel Chat Clear
	// Channel Chat Clear User Messages
	// Channel Chat Message
	// Channel Chat Message Delete
	// Channel Chat Notification
	// Channel Chat Settings Update
	ChannelBot Scope = "channel:bot"

	// Manage ads schedule on a channel.
	// API
	// Snooze Next Ad
	ChannelManageAds Scope = "channel:manage:ads"

	// Read the ads schedule and details on your channel.
	// API
	// Get Ad Schedule
	// EventSub
	// Channel Ad Break Begin
	ChannelReadAds Scope = "channel:read:ads"

	// Manage a channel's broadcast configuration, including updating channel configuration and managing stream markers and stream tags.
	// API
	// Modify Channel Information
	// Create Stream Marker
	// Replace Stream Tags
	ChannelManageBroadcast Scope = "channel:manage:broadcast"

	// Read charity campaign details and user donations on your channel.
	// API
	// Get Charity Campaign
	// Get Charity Campaign Donations
	// EventSub
	// Charity Donation
	// Charity Campaign Start
	// Charity Campaign Progress
	// Charity Campaign Stop
	ChannelReadCharity Scope = "channel:read:charity"

	// Manage Clips for a channel.
	// API
	// Create Clip From VOD
	// Get Clips Download
	ChannelManageClips Scope = "channel:manage:clips"

	// Run commercials on a channel.
	// API
	// Start Commercial
	ChannelEditCommercial Scope = "channel:edit:commercial"

	// View a list of users with the editor role for a channel.
	// API
	// Get Channel Editors
	ChannelReadEditors Scope = "channel:read:editors"

	// Manage a channel's Extension configuration, including activating Extensions.
	// API
	// Get User Active Extensions
	// Update User Extensions
	ChannelManageExtensions Scope = "channel:manage:extensions"

	// View Creator Goals for a channel.
	// API
	// Get Creator Goals
	// EventSub
	// Goal Begin
	// Goal Progress
	// Goal End
	ChannelReadGoals Scope = "channel:read:goals"

	// Read Guest Star details for your channel.
	// API
	// Get Channel Guest Star Settings
	// Get Guest Star Session
	// Get Guest Star Invites
	// EventSub
	// Channel Guest Star Session Begin
	// Channel Guest Star Session End
	// Channel Guest Star Guest Update
	// Channel Guest Star Settings Update
	ChannelReadGuestStar Scope = "channel:read:guest_star"

	// Manage Guest Star for your channel.
	// API
	// Update Channel Guest Star Settings
	// Create Guest Star Session
	// End Guest Star Session
	// Send Guest Star Invite
	// Delete Guest Star Invite
	// Assign Guest Star Slot
	// Update Guest Star Slot
	// Delete Guest Star Slot
	// Update Guest Star Slot Settings
	// EventSub
	// Channel Guest Star Session Begin
	// Channel Guest Star Session End
	// Channel Guest Star Guest Update
	// Channel Guest Star Settings Update
	ChannelManageGuestStar Scope = "channel:manage:guest_star"

	// View Hype Train information for a channel.
	// API
	// Get Hype Train Status
	// EventSub
	// Hype Train Begin
	// Hype Train Progress
	// Hype Train End
	ChannelReadHypeTrain Scope = "channel:read:hype_train"

	// Add or remove the moderator role from users in your channel.
	// API
	// Add Channel Moderator
	// Remove Channel Moderator
	// Get Moderators
	ChannelManageModerators Scope = "channel:manage:moderators"

	// View a channel's polls.
	// API
	// Get Polls
	// EventSub
	// Channel Poll Begin
	// Channel Poll Progress
	// Channel Poll End
	ChannelReadPolls Scope = "channel:read:polls"

	// Manage a channel's polls.
	// API
	// Get Polls
	// Create Poll
	// End Poll
	// EventSub
	// Channel Poll Begin
	// Channel Poll Progress
	// Channel Poll End
	ChannelManagePolls Scope = "channel:manage:polls"

	// View a channel's Channel Points Predictions.
	// API
	// Get Channel Points Predictions
	// EventSub
	// Channel Prediction Begin
	// Channel Prediction Progress
	// Channel Prediction Lock
	// Channel Prediction End
	ChannelReadPredictions Scope = "channel:read:predictions"

	// Manage of channel's Channel Points Predictions.
	// API
	// Get Channel Points Predictions
	// Create Channel Points Prediction
	// End Channel Points Prediction
	// EventSub
	// Channel Prediction Begin
	// Channel Prediction Progress
	// Channel Prediction Lock
	// Channel Prediction End
	ChannelManagePredictions Scope = "channel:manage:predictions"

	// Manage a channel raiding another channel.
	// API
	// Start a raid
	// Cancel a raid
	ChannelManageRaids Scope = "channel:manage:raids"

	// View Channel Points custom rewards and their redemptions on a channel.
	// API
	// Get Custom Reward
	// Get Custom Reward Redemption
	// EventSub
	// Channel Points Automatic Reward Redemption
	// Channel Points Automatic Reward Redemption v2
	// Channel Points Custom Reward Add
	// Channel Points Custom Reward Update
	// Channel Points Custom Reward Remove
	// Channel Points Custom Reward Redemption Add
	// Channel Points Custom Reward Redemption Update
	ChannelReadRedemptions Scope = "channel:read:redemptions"

	// Manage Channel Points custom rewards and their redemptions on a channel.
	// API
	// Get Custom Reward
	// Get Custom Reward Redemption
	// Create Custom Rewards
	// Delete Custom Reward
	// Update Custom Reward
	// Update Redemption Status
	// EventSub
	// Channel Points Automatic Reward Redemption
	// Channel Points Custom Reward Add
	// Channel Points Custom Reward Update
	// Channel Points Custom Reward Remove
	// Channel Points Custom Reward Redemption Add
	// Channel Points Custom Reward Redemption Update
	ChannelManageRedemptions Scope = "channel:manage:redemptions"

	// Manage a channel's stream schedule.
	// API
	// Update Channel Stream Schedule
	// Create Channel Stream Schedule Segment
	// Update Channel Stream Schedule Segment
	// Delete Channel Stream Schedule Segment
	ChannelManageSchedule Scope = "channel:manage:schedule"

	// View an authorized user's stream key.
	// API
	// Get Stream Key
	ChannelReadStreamKey Scope = "channel:read:stream_key"

	// View a list of all subscribers to a channel and check if a user is subscribed to a channel.
	// API
	// Get Broadcaster Subscriptions
	// EventSub
	// Channel Subscribe
	// Channel Subscription End
	// Channel Subscription Gift
	// Channel Subscription Message
	ChannelReadSubscriptions Scope = "channel:read:subscriptions"

	// Manage a channel's videos, including deleting videos.
	// API
	// Delete Videos
	ChannelManageVideos Scope = "channel:manage:videos"

	// Read the list of VIPs in your channel.
	// API
	// Get VIPs
	// EventSub
	// Channel VIP Add
	// Channel VIP Remove
	ChannelReadVIPs Scope = "channel:read:vips"

	// Add or remove the VIP role from users in your channel.
	// API
	// Get VIPs
	// Add Channel VIP
	// Remove Channel VIP
	// EventSub
	// Channel VIP Add
	// Channel VIP Remove
	ChannelManageVIPs Scope = "channel:manage:vips"

	// Perform moderation actions in a channel.
	// EventSub
	// Channel Ban
	// Channel Unban
	ChannelModerate Scope = "channel:moderate"

	// Manage Clips for a channel.
	// API
	// Create Clip
	ClipsEdit Scope = "clips:edit"

	// Manage Clips as an editor.
	// API
	// Create Clip From VOD
	// Get Clips Download
	EditorManageClips Scope = "editor:manage:clips"

	// View a channel's moderation data including Moderators, Bans, Timeouts, and Automod settings.
	// API
	// Check AutoMod Status
	// Get Banned Users
	// Get Moderators
	// EventSub
	// Channel Moderator Add
	// Channel Moderator Remove
	ModerationRead Scope = "moderation:read"

	// Send announcements in channels where you have the moderator role.
	// API
	// Send Chat Announcement
	ModeratorManageAnnouncements Scope = "moderator:manage:announcements"

	// Manage messages held for review by AutoMod in channels where you are a moderator.
	// API
	// Manage Held AutoMod Messages
	// EventSub
	// AutoMod Message Hold
	// AutoMod Message Hold v2
	// AutoMod Message Update
	// AutoMod Message Update v2
	// AutoMod Terms Update
	ModeratorManageAutomod Scope = "moderator:manage:automod"

	// View a broadcaster's AutoMod settings.
	// API
	// Get AutoMod Settings
	// EventSub
	// AutoMod Settings Update
	ModeratorReadAutomodSettings Scope = "moderator:read:automod_settings"

	// Manage a broadcaster's AutoMod settings.
	// API
	// Update AutoMod Settings
	ModeratorManageAutomodSettings Scope = "moderator:manage:automod_settings"

	// Read the list of bans or unbans in channels where you have the moderator role.
	// EventSub
	// Channel Moderate
	// Channel Moderate v2
	ModeratorReadBannedUsers Scope = "moderator:read:banned_users"

	// Ban and unban users.
	// API
	// Get Banned Users
	// Ban User
	// Unban User
	// EventSub
	// Channel Moderate
	// Channel Moderate v2
	ModeratorManageBannedUsers Scope = "moderator:manage:banned_users"

	// View a broadcaster's list of blocked terms.
	// API
	// Get Blocked Terms
	// EventSub
	// Channel Moderate
	ModeratorReadBlockedTerms Scope = "moderator:read:blocked_terms"

	// Manage a broadcaster's list of blocked terms.
	// API
	// Get Blocked Terms
	// Add Blocked Term
	// Remove Blocked Term
	// EventSub
	// Channel Moderate
	ModeratorManageBlockedTerms Scope = "moderator:manage:blocked_terms"

	// Read deleted chat messages in channels where you have the moderator role and get pinned chat messages.
	// API
	// Get Pinned Chat Message
	// EventSub
	// Channel Moderate
	ModeratorReadChatMessages Scope = "moderator:read:chat_messages"

	// Delete chat messages in channels where you have the moderator role and manage pinned chat messages.
	// API
	// Delete Chat Messages
	// Pin Chat Message
	// Update Pinned Chat Message
	// Unpin Chat Message
	// EventSub
	// Channel Moderate
	ModeratorManageChatMessages Scope = "moderator:manage:chat_messages"

	// View a broadcaster's chat room settings.
	// API
	// Get Chat Settings
	// EventSub
	// Channel Moderate
	ModeratorReadChatSettings Scope = "moderator:read:chat_settings"

	// Manage a broadcaster's chat room settings.
	// API
	// Update Chat Settings
	// EventSub
	// Channel Moderate
	ModeratorManageChatSettings Scope = "moderator:manage:chat_settings"

	// View the chatters in a broadcaster's chat room.
	// API
	// Get Chatters
	ModeratorReadChatters Scope = "moderator:read:chatters"

	// Read the followers of a broadcaster.
	// API
	// Get Channel Followers
	// EventSub
	// Channel Follow
	ModeratorReadFollowers Scope = "moderator:read:followers"

	// Read Guest Star details for channels where you are a Guest Star moderator.
	// API
	// Get Channel Guest Star Settings
	// Get Guest Star Session
	// Get Guest Star Invites
	// EventSub
	// Channel Guest Star Session Begin
	// Channel Guest Star Session End
	// Channel Guest Star Guest Update
	// Channel Guest Star Settings Update
	ModeratorReadGuestStar Scope = "moderator:read:guest_star"

	// Manage Guest Star for channels where you are a Guest Star moderator.
	// API
	// Send Guest Star Invite
	// Delete Guest Star Invite
	// Assign Guest Star Slot
	// Update Guest Star Slot
	// Delete Guest Star Slot
	// Update Guest Star Slot Settings
	// EventSub
	// Channel Guest Star Session Begin
	// Channel Guest Star Session End
	// Channel Guest Star Guest Update
	// Channel Guest Star Settings Update
	ModeratorManageGuestStar Scope = "moderator:manage:guest_star"

	// Read the list of moderators in channels where you have the moderator role.
	// EventSub
	// Channel Moderate
	// Channel Moderate v2
	ModeratorReadModerators Scope = "moderator:read:moderators"

	// View a broadcaster's Shield Mode status.
	// API
	// Get Shield Mode Status
	// EventSub
	// Shield Mode Begin
	// Shield Mode End
	ModeratorReadShieldMode Scope = "moderator:read:shield_mode"

	// Manage a broadcaster's Shield Mode status.
	// API
	// Update Shield Mode Status
	// EventSub
	// Shield Mode Begin
	// Shield Mode End
	ModeratorManageShieldMode Scope = "moderator:manage:shield_mode"

	// View a broadcaster's shoutouts.
	// EventSub
	// Shoutout Create
	// Shoutout Received
	ModeratorReadShoutouts Scope = "moderator:read:shoutouts"

	// Manage a broadcaster's shoutouts.
	// API
	// Send a Shoutout
	// EventSub
	// Shoutout Create
	// Shoutout Received
	ModeratorManageShoutouts Scope = "moderator:manage:shoutouts"

	// Read chat messages from suspicious users and see users flagged as suspicious in channels where the user has the moderator role.
	// EventSub
	// Channel Suspicious User Message
	// Channel Suspicious User Update
	ModeratorReadSuspiciousUsers Scope = "moderator:read:suspicious_users"

	// Manage suspicious user statuses in channels where the user has the moderator role.
	// API
	// Add suspicious status to chat user
	// Remove suspicious status from chat user
	ModeratorManageSuspiciousUsers Scope = "moderator:manage:suspicious_users"

	// View a broadcaster's unban requests.
	// API
	// Get Unban Requests
	// EventSub
	// Channel Unban Request Create
	// Channel Unban Request Resolve
	// Channel Moderate
	ModeratorReadUnbanRequests Scope = "moderator:read:unban_requests"

	// Manage a broadcaster's unban requests.
	// API
	// Resolve Unban Requests
	// EventSub
	// Channel Unban Request Create
	// Channel Unban Request Resolve
	// Channel Moderate
	ModeratorManageUnbanRequests Scope = "moderator:manage:unban_requests"

	// Read the list of VIPs in channels where you have the moderator role.
	// EventSub
	// Channel Moderate
	// Channel Moderate v2
	ModeratorReadVIPs Scope = "moderator:read:vips"

	// Read warnings in channels where you have the moderator role.
	// EventSub
	// Channel Moderate v2
	// Channel Warning Acknowledge
	// Channel Warning Send
	ModeratorReadWarnings Scope = "moderator:read:warnings"

	// Warn users in channels where you have the moderator role.
	// API
	// Warn Chat User
	// EventSub
	// Channel Moderate v2
	// Channel Warning Acknowledge
	// Channel Warning Send
	ModeratorManageWarnings Scope = "moderator:manage:warnings"

	// Join a specified chat channel as your user and appear as a bot, and perform chat-related actions as your user.
	// API
	// Send Chat Message
	// EventSub
	// Channel Chat Clear
	// Channel Chat Clear User Messages
	// Channel Chat Message
	// Channel Chat Message Delete
	// Channel Chat Notification
	// Channel Chat Settings Update
	// Channel Chat User Message Hold
	// Channel Chat User Message Update
	UserBot Scope = "user:bot"

	// Manage a user object.
	// API
	// Update User
	UserEdit Scope = "user:edit"

	// View and edit a user's broadcasting configuration, including Extension configurations.
	// API
	// Get User Extensions
	// Get User Active Extensions
	// Update User Extensions
	UserEditBroadcast Scope = "user:edit:broadcast"

	// View the block list of a user.
	// API
	// Get User Block List
	UserReadBlockedUsers Scope = "user:read:blocked_users"

	// Manage the block list of a user.
	// API
	// Block User
	// Unblock User
	UserManageBlockedUsers Scope = "user:manage:blocked_users"

	// View a user's broadcasting configuration, including Extension configurations.
	// API
	// Get Stream Markers
	// Get User Extensions
	// Get User Active Extensions
	UserReadBroadcast Scope = "user:read:broadcast"

	// Receive chatroom messages and informational notifications relating to a channel's chatroom.
	// EventSub
	// Channel Chat Clear
	// Channel Chat Clear User Messages
	// Channel Chat Message
	// Channel Chat Message Delete
	// Channel Chat Notification
	// Channel Chat Settings Update
	// Channel Chat User Message Hold
	// Channel Chat User Message Update
	UserReadChat Scope = "user:read:chat"

	// Update the color used for the user's name in chat.
	// API
	// Update User Chat Color
	UserManageChatColor Scope = "user:manage:chat_color"

	// View a user's email address.
	// API
	// Get Users (optional)
	// Update User (optional)
	// EventSub
	// User Update (optional)
	UserReadEmail Scope = "user:read:email"

	// View emotes available to a user.
	// API
	// Get User Emotes
	UserReadEmotes Scope = "user:read:emotes"

	// View the list of channels a user follows.
	// API
	// Get Followed Channels
	// Get Followed Streams
	UserReadFollows Scope = "user:read:follows"

	// Read the list of channels you have moderator privileges in.
	// API
	// Get Moderated Channels
	UserReadModeratedChannels Scope = "user:read:moderated_channels"

	// View if an authorized user is subscribed to specific channels.
	// API
	// Check User Subscription
	UserReadSubscriptions Scope = "user:read:subscriptions"

	// Receive whispers sent to your user.
	// EventSub
	// Whisper Received
	UserReadWhispers Scope = "user:read:whispers"

	// Receive whispers sent to your user, and send whispers on your user's behalf.
	// API
	// Send Whisper
	// EventSub
	// Whisper Received
	UserManageWhispers Scope = "user:manage:whispers"

	// Send chat messages to a chatroom.
	// API
	// Send Chat Message
	UserWriteChat Scope = "user:write:chat"

	// IRC SCOPES
	// Send chat messages to a chatroom using an IRC connection.
	ChatEdit Scope = "chat:edit"
	// View chat messages sent in a chatroom using an IRC connection.
	ChatRead Scope = "chat:read"

	// PUB SUB SCOPES
	// Receive whisper messages for your user using PubSub.
	WhispersRead Scope = "whispers:read"
)
