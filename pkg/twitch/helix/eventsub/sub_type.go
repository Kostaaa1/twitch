package eventsub

type subType string

const (
	// AutomodMessageHold - A user is notified if a message is caught by automod for review. (v1)
	AutomodMessageHold subType = "automod.message.hold"
	// AutomodMessageHoldV2 - A user is notified if a message is caught by automod for review. Only public blocked terms trigger notifications, not private ones. (v2)
	AutomodMessageHoldV2 subType = "automod.message.hold"
	// AutomodMessageUpdate - A message in the automod queue had its status changed. (v1)
	AutomodMessageUpdate subType = "automod.message.update"
	// AutomodMessageUpdateV2 - A message in the automod queue had its status changed. Only public blocked terms trigger notifications, not private ones. (v2)
	AutomodMessageUpdateV2 subType = "automod.message.update"
	// AutomodSettingsUpdate - A notification is sent when a broadcaster's automod settings are updated. (v1)
	AutomodSettingsUpdate subType = "automod.settings.update"
	// AutomodTermsUpdate - A notification is sent when a broadcaster's automod terms are updated. Changes to private terms are not sent. (v1)
	AutomodTermsUpdate subType = "automod.terms.update"

	// ChannelBitsUse - A notification is sent whenever Bits are used on a channel. (v1)
	ChannelBitsUse subType = "channel.bits.use"
	// ChannelUpdate - A broadcaster updates their channel properties e.g., category, title, content classification labels, broadcast, or language. (v2)
	ChannelUpdate subType = "channel.update"
	// ChannelFollow - A specified channel receives a follow. (v2)
	ChannelFollow subType = "channel.follow"
	// ChannelAdBreakBegin - A midroll commercial break has started running. (v1)
	ChannelAdBreakBegin subType = "channel.ad_break.begin"

	// ChannelChatClear - A moderator or bot has cleared all messages from the chat room. (v1)
	ChannelChatClear subType = "channel.chat.clear"
	// ChannelChatClearUserMessages - A moderator or bot has cleared all messages from a specific user. (v1)
	ChannelChatClearUserMessages subType = "channel.chat.clear_user_messages"
	// ChannelChatMessage - Any user sends a message to a specific chat room. (v1)
	ChannelChatMessage subType = "channel.chat.message"
	// ChannelChatMessageDelete - A moderator has removed a specific message. (v1)
	ChannelChatMessageDelete subType = "channel.chat.message_delete"
	// ChannelChatNotification - A notification for when an event that appears in chat has occurred. (v1)
	ChannelChatNotification subType = "channel.chat.notification"
	// ChannelChatSettingsUpdate - A notification for when a broadcaster's chat settings are updated. (v1)
	ChannelChatSettingsUpdate subType = "channel.chat_settings.update"
	// ChannelChatUserMessageHold - A user is notified if their message is caught by automod. (v1)
	ChannelChatUserMessageHold subType = "channel.chat.user_message_hold"
	// ChannelChatUserMessageUpdate - A user is notified if their message's automod status is updated. (v1)
	ChannelChatUserMessageUpdate subType = "channel.chat.user_message_update"

	// ChannelSharedChatSessionBegin - A notification when a channel becomes active in an active shared chat session. (v1)
	ChannelSharedChatSessionBegin subType = "channel.shared_chat.begin"
	// ChannelSharedChatSessionUpdate - A notification when the active shared chat session the channel is in changes. (v1)
	ChannelSharedChatSessionUpdate subType = "channel.shared_chat.update"
	// ChannelSharedChatSessionEnd - A notification when a channel leaves a shared chat session or the session ends. (v1)
	ChannelSharedChatSessionEnd subType = "channel.shared_chat.end"

	// ChannelSubscribe - A notification is sent when a specified channel receives a subscriber. This does not include resubscribes. (v1)
	ChannelSubscribe subType = "channel.subscribe"
	// ChannelSubscriptionEnd - A notification when a subscription to the specified channel ends. (v1)
	ChannelSubscriptionEnd subType = "channel.subscription.end"
	// ChannelSubscriptionGift - A notification when a viewer gives a gift subscription to one or more users in the specified channel. (v1)
	ChannelSubscriptionGift subType = "channel.subscription.gift"
	// ChannelSubscriptionMessage - A notification when a user sends a resubscription chat message in a specific channel. (v1)
	ChannelSubscriptionMessage subType = "channel.subscription.message"
	// ChannelCheer - A user cheers on the specified channel. (v1)
	ChannelCheer subType = "channel.cheer"
	// ChannelRaid - A broadcaster raids another broadcaster's channel. (v1)
	ChannelRaid subType = "channel.raid"
	// ChannelBan - A viewer is banned from the specified channel. (v1)
	ChannelBan subType = "channel.ban"
	// ChannelUnban - A viewer is unbanned from the specified channel. (v1)
	ChannelUnban subType = "channel.unban"
	// ChannelUnbanRequestCreate - A user creates an unban request. (v1)
	ChannelUnbanRequestCreate subType = "channel.unban_request.create"
	// ChannelUnbanRequestResolve - An unban request has been resolved. (v1)
	ChannelUnbanRequestResolve subType = "channel.unban_request.resolve"

	// ChannelModerate - A moderator performs a moderation action in a channel. (v1)
	ChannelModerate subType = "channel.moderate"
	// ChannelModerateV2 - A moderator performs a moderation action in a channel. Includes warnings. (v2)
	ChannelModerateV2 subType = "channel.moderate"
	// ChannelModeratorAdd - Moderator privileges were added to a user on a specified channel. (v1)
	ChannelModeratorAdd subType = "channel.moderator.add"
	// ChannelModeratorRemove - Moderator privileges were removed from a user on a specified channel. (v1)
	ChannelModeratorRemove subType = "channel.moderator.remove"

	// ChannelGuestStarSessionBegin - The host began a new Guest Star session. (beta)
	ChannelGuestStarSessionBegin subType = "channel.guest_star_session.begin"
	// ChannelGuestStarSessionEnd - A running Guest Star session has ended. (beta)
	ChannelGuestStarSessionEnd subType = "channel.guest_star_session.end"
	// ChannelGuestStarGuestUpdate - A guest or a slot is updated in an active Guest Star session. (beta)
	ChannelGuestStarGuestUpdate subType = "channel.guest_star_guest.update"
	// ChannelGuestStarSettingsUpdate - The host preferences for Guest Star have been updated. (beta)
	ChannelGuestStarSettingsUpdate subType = "channel.guest_star_settings.update"

	// ChannelPointsAutomaticRewardRedemptionAdd - A viewer has redeemed an automatic channel points reward on the specified channel. (v1)
	ChannelPointsAutomaticRewardRedemptionAdd subType = "channel.channel_points_automatic_reward_redemption.add"
	// ChannelPointsAutomaticRewardRedemptionAddV2 - A viewer has redeemed an automatic channel points reward on the specified channel. (v2)
	ChannelPointsAutomaticRewardRedemptionAddV2 subType = "channel.channel_points_automatic_reward_redemption.add"
	// ChannelPointsCustomRewardAdd - A custom channel points reward has been created for the specified channel. (v1)
	ChannelPointsCustomRewardAdd subType = "channel.channel_points_custom_reward.add"
	// ChannelPointsCustomRewardUpdate - A custom channel points reward has been updated for the specified channel. (v1)
	ChannelPointsCustomRewardUpdate subType = "channel.channel_points_custom_reward.update"
	// ChannelPointsCustomRewardRemove - A custom channel points reward has been removed from the specified channel. (v1)
	ChannelPointsCustomRewardRemove subType = "channel.channel_points_custom_reward.remove"
	// ChannelPointsCustomRewardRedemptionAdd - A viewer has redeemed a custom channel points reward on the specified channel. (v1)
	ChannelPointsCustomRewardRedemptionAdd subType = "channel.channel_points_custom_reward_redemption.add"
	// ChannelPointsCustomRewardRedemptionUpdate - A redemption of a channel points custom reward has been updated for the specified channel. (v1)
	ChannelPointsCustomRewardRedemptionUpdate subType = "channel.channel_points_custom_reward_redemption.update"
	// ChannelCustomPowerUpRedemptionAdd - A viewer has redeemed a custom Power-up on the specified channel. (v1)
	ChannelCustomPowerUpRedemptionAdd subType = "channel.custom_power_up_redemption.add"

	// ChannelPollBegin - A poll started on a specified channel. (v1)
	ChannelPollBegin subType = "channel.poll.begin"
	// ChannelPollProgress - Users respond to a poll on a specified channel. (v1)
	ChannelPollProgress subType = "channel.poll.progress"
	// ChannelPollEnd - A poll ended on a specified channel. (v1)
	ChannelPollEnd subType = "channel.poll.end"

	// ChannelPredictionBegin - A Prediction started on a specified channel. (v1)
	ChannelPredictionBegin subType = "channel.prediction.begin"
	// ChannelPredictionProgress - Users participated in a Prediction on a specified channel. (v1)
	ChannelPredictionProgress subType = "channel.prediction.progress"
	// ChannelPredictionLock - A Prediction was locked on a specified channel. (v1)
	ChannelPredictionLock subType = "channel.prediction.lock"
	// ChannelPredictionEnd - A Prediction ended on a specified channel. (v1)
	ChannelPredictionEnd subType = "channel.prediction.end"

	// ChannelSuspiciousUserMessage - A chat message has been sent by a suspicious user. (v1)
	ChannelSuspiciousUserMessage subType = "channel.suspicious_user.message"
	// ChannelSuspiciousUserUpdate - A suspicious user has been updated. (v1)
	ChannelSuspiciousUserUpdate subType = "channel.suspicious_user.update"

	// ChannelVIPAdd - A VIP is added to the channel. (v1)
	ChannelVIPAdd subType = "channel.vip.add"
	// ChannelVIPRemove - A VIP is removed from the channel. (v1)
	ChannelVIPRemove subType = "channel.vip.remove"

	// ChannelWarningAcknowledge - A user acknowledges a warning. Broadcasters and moderators can see the warning's details. (v1)
	ChannelWarningAcknowledge subType = "channel.warning.acknowledge"
	// ChannelWarningSend - A user is sent a warning. Broadcasters and moderators can see the warning's details. (v1)
	ChannelWarningSend subType = "channel.warning.send"

	// CharityDonation - Sends an event notification when a user donates to the broadcaster's charity campaign. (v1)
	CharityDonation subType = "channel.charity_campaign.donate"
	// CharityCampaignStart - Sends an event notification when the broadcaster starts a charity campaign. (v1)
	CharityCampaignStart subType = "channel.charity_campaign.start"
	// CharityCampaignProgress - Sends an event notification when progress is made towards the campaign's goal or when the broadcaster changes the fundraising goal. (v1)
	CharityCampaignProgress subType = "channel.charity_campaign.progress"
	// CharityCampaignStop - Sends an event notification when the broadcaster stops a charity campaign. (v1)
	CharityCampaignStop subType = "channel.charity_campaign.stop"

	// ConduitShardDisabled - Sends a notification when EventSub disables a shard due to the status of the underlying transport changing. (v1)
	ConduitShardDisabled subType = "conduit.shard.disabled"

	// DropEntitlementGrant - An entitlement for a Drop is granted to a user. (v1)
	DropEntitlementGrant subType = "drop.entitlement.grant"

	// ExtensionBitsTransactionCreate - A Bits transaction occurred for a specified Twitch Extension. (v1)
	ExtensionBitsTransactionCreate subType = "extension.bits_transaction.create"

	// GoalBegin - Get notified when a broadcaster begins a goal. (v1)
	GoalBegin subType = "channel.goal.begin"
	// GoalProgress - Get notified when progress (either positive or negative) is made towards a broadcaster's goal. (v1)
	GoalProgress subType = "channel.goal.progress"
	// GoalEnd - Get notified when a broadcaster ends a goal. (v1)
	GoalEnd subType = "channel.goal.end"

	// HypeTrainBegin - A Hype Train begins on the specified channel. (v2)
	HypeTrainBegin subType = "channel.hype_train.begin"
	// HypeTrainProgress - A Hype Train makes progress on the specified channel. (v2)
	HypeTrainProgress subType = "channel.hype_train.progress"
	// HypeTrainEnd - A Hype Train ends on the specified channel. (v2)
	HypeTrainEnd subType = "channel.hype_train.end"

	// ShieldModeBegin - Sends a notification when the broadcaster activates Shield Mode. (v1)
	ShieldModeBegin subType = "channel.shield_mode.begin"
	// ShieldModeEnd - Sends a notification when the broadcaster deactivates Shield Mode. (v1)
	ShieldModeEnd subType = "channel.shield_mode.end"

	// ShoutoutCreate - Sends a notification when the specified broadcaster sends a Shoutout. (v1)
	ShoutoutCreate subType = "channel.shoutout.create"
	// ShoutoutReceived - Sends a notification when the specified broadcaster receives a Shoutout. (v1)
	ShoutoutReceived subType = "channel.shoutout.receive"

	// StreamOnline - The specified broadcaster starts a stream. (v1)
	StreamOnline subType = "stream.online"
	// StreamOffline - The specified broadcaster stops a stream. (v1)
	StreamOffline subType = "stream.offline"

	// UserAuthorizationGrant - A user's authorization has been granted to your client id. (v1)
	UserAuthorizationGrant subType = "user.authorization.grant"
	// UserAuthorizationRevoke - A user's authorization has been revoked for your client id. (v1)
	UserAuthorizationRevoke subType = "user.authorization.revoke"
	// UserUpdate - A user has updated their account. (v1)
	UserUpdate subType = "user.update"
	// WhisperReceived - A user receives a whisper. (v1)
	WhisperReceived subType = "user.whisper.message"
)
