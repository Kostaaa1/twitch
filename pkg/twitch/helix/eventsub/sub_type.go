type SubType string

var (
// Automod Message Hold
// j user is notified if a message is caught by automod for review.
// AutomodMessageHold SubType = "automod.message.hold"
// // Automod Message Hold V2
// // A user is notified if a message is caught by automod for review. Only public blocked terms trigger notifications, not private ones.
// AutomodMessageHold SubType = "automod.message.hold"
// Automod Message Update
// AutomodMessageHold SubType = "automod.message.update"
//
//	1	A message in the automod queue had its status changed.
//
// Automod Message Update V2
// AutomodMessageHold SubType = "automod.message.update"
//
//	2	A message in the automod queue had its status changed. Only public blocked terms trigger notifications, not private ones.
//
// Automod Settings Update	automod.settings.update	1	A notification is sent when a broadcaster’s automod settings are updated.
// Automod Terms Update	automod.terms.update	1	A notification is sent when a broadcaster’s automod terms are updated. Changes to private terms are not sent.
// Channel Bits Use
// NEW	channel.bits.use	1	A notification is sent whenever Bits are used on a channel.
// Channel Update	channel.update	2	A broadcaster updates their channel properties e.g., category, title, content classification labels, broadcast, or language.
// Channel Follow	channel.follow	2	A specified channel receives a follow.
// Channel Ad Break Begin	channel.ad_break.begin	1	A midroll commercial break has started running.
// Channel Chat Clear	channel.chat.clear	1	A moderator or bot has cleared all messages from the chat room.
// Channel Chat Clear User Messages	channel.chat.clear_user_messages	1	A moderator or bot has cleared all messages from a specific user.
// Channel Chat Message	channel.chat.message	1	Any user sends a message to a specific chat room.
// Channel Chat Message Delete	channel.chat.message_delete	1	A moderator has removed a specific message.
// Channel Chat Notification	channel.chat.notification	1	A notification for when an event that appears in chat has occurred.
// Channel Chat Settings Update	channel.chat_settings.update	1	A notification for when a broadcaster’s chat settings are updated.
// Channel Chat User Message Hold	channel.chat.user_message_hold	1	A user is notified if their message is caught by automod.
// Channel Chat User Message Update	channel.chat.user_message_update	1	A user is notified if their message’s automod status is updated.
// Channel Shared Chat Session Begin	channel.shared_chat.begin	1	A notification when a channel becomes active in an active shared chat session.
// Channel Shared Chat Session Update	channel.shared_chat.update	1	A notification when the active shared chat session the channel is in changes.
// Channel Shared Chat Session End	channel.shared_chat.end	1	A notification when a channel leaves a shared chat session or the session ends.
// Channel Subscribe	channel.subscribe	1	A notification is sent when a specified channel receives a subscriber. This does not include resubscribes.
// Channel Subscription End	channel.subscription.end	1	A notification when a subscription to the specified channel ends.
// Channel Subscription Gift	channel.subscription.gift	1	A notification when a viewer gives a gift subscription to one or more users in the specified channel.
// Channel Subscription Message	channel.subscription.message	1	A notification when a user sends a resubscription chat message in a specific channel.
// Channel Cheer	channel.cheer	1	A user cheers on the specified channel.
// Channel Raid	channel.raid	1	A broadcaster raids another broadcaster’s channel.
// Channel Ban	channel.ban	1	A viewer is banned from the specified channel.
// Channel Unban	channel.unban	1	A viewer is unbanned from the specified channel.
// Channel Unban Request Create	channel.unban_request.create 	1	A user creates an unban request.
// Channel Unban Request Resolve	channel.unban_request.resolve 	1	An unban request has been resolved.
// Channel Moderate	channel.moderate	1	A moderator performs a moderation action in a channel.
// Channel Moderate V2	channel.moderate	2	A moderator performs a moderation action in a channel. Includes warnings.
// Channel Moderator Add	channel.moderator.add	1	Moderator privileges were added to a user on a specified channel.
// Channel Moderator Remove	channel.moderator.remove	1	Moderator privileges were removed from a user on a specified channel.
// Channel Guest Star Session Begin
// BETA	channel.guest_star_session.begin	beta	The host began a new Guest Star session.
// Channel Guest Star Session End
// BETA	channel.guest_star_session.end	beta	A running Guest Star session has ended.
// Channel Guest Star Guest Update
// BETA	channel.guest_star_guest.update	beta	A guest or a slot is updated in an active Guest Star session.
// Channel Guest Star Settings Update
// BETA	channel.guest_star_settings.update	beta	The host preferences for Guest Star have been updated.
// Channel Points Automatic Reward Redemption Add	channel.channel_points_automatic_reward_redemption.add	1	A viewer has redeemed an automatic channel points reward on the specified channel.
// Channel Points Automatic Reward Redemption Add V2	channel.channel_points_automatic_reward_redemption.add	2	A viewer has redeemed an automatic channel points reward on the specified channel.
// Channel Points Custom Reward Add	channel.channel_points_custom_reward.add	1	A custom channel points reward has been created for the specified channel.
// Channel Points Custom Reward Update	channel.channel_points_custom_reward.update	1	A custom channel points reward has been updated for the specified channel.
// Channel Points Custom Reward Remove	channel.channel_points_custom_reward.remove	1	A custom channel points reward has been removed from the specified channel.
// Channel Points Custom Reward Redemption Add	channel.channel_points_custom_reward_redemption.add	1	A viewer has redeemed a custom channel points reward on the specified channel.
// Channel Points Custom Reward Redemption Update	channel.channel_points_custom_reward_redemption.update	1	A redemption of a channel points custom reward has been updated for the specified channel.
// Channel Custom Power-ups Redemption Add
// NEW	channel.custom_power_up_redemption.add	1	A viewer has redeemed a custom Power-up on the specified channel.
// Channel Poll Begin	channel.poll.begin	1	A poll started on a specified channel.
// Channel Poll Progress	channel.poll.progress	1	Users respond to a poll on a specified channel.
// Channel Poll End	channel.poll.end	1	A poll ended on a specified channel.
// Channel Prediction Begin	channel.prediction.begin	1	A Prediction started on a specified channel.
// Channel Prediction Progress	channel.prediction.progress	1	Users participated in a Prediction on a specified channel.
// Channel Prediction Lock	channel.prediction.lock	1	A Prediction was locked on a specified channel.
// Channel Prediction End	channel.prediction.end	1	A Prediction ended on a specified channel.
// Channel Suspicious User Message
// NEW	channel.suspicious_user.message	1	A chat message has been sent by a suspicious user.
// Channel Suspicious User Update
// NEW	channel.suspicious_user.update	1	A suspicious user has been updated.
// Channel VIP Add	channel.vip.add	1	A VIP is added to the channel.
// Channel VIP Remove	channel.vip.remove	1	A VIP is removed from the channel.
// Channel Warning Acknowledgement	channel.warning.acknowledge	1	A user acknowledges a warning. Broadcasters and moderators can see the warning’s details.
// Channel Warning Send
// NEW	channel.warning.send	1	A user is sent a warning. Broadcasters and moderators can see the warning’s details.
// Charity Donation	channel.charity_campaign.donate	1	Sends an event notification when a user donates to the broadcaster’s charity campaign.
// Charity Campaign Start	channel.charity_campaign.start	1	Sends an event notification when the broadcaster starts a charity campaign.
// Charity Campaign Progress	channel.charity_campaign.progress	1	Sends an event notification when progress is made towards the campaign’s goal or when the broadcaster changes the fundraising goal.
// Charity Campaign Stop	channel.charity_campaign.stop	1	Sends an event notification when the broadcaster stops a charity campaign.
// Conduit Shard Disabled
// NEW	conduit.shard.disabled	1	Sends a notification when EventSub disables a shard due to the status of the underlying transport changing.
// Drop Entitlement Grant	drop.entitlement.grant	1	An entitlement for a Drop is granted to a user.
// Extension Bits Transaction Create	extension.bits_transaction.create	1	A Bits transaction occurred for a specified Twitch Extension.
// Goal Begin	channel.goal.begin	1	Get notified when a broadcaster begins a goal.
// Goal Progress	channel.goal.progress	1	Get notified when progress (either positive or negative) is made towards a broadcaster’s goal.
// Goal End	channel.goal.end	1	Get notified when a broadcaster ends a goal.
// Hype Train Begin	channel.hype_train.begin	2	A Hype Train begins on the specified channel.
// Hype Train Progress	channel.hype_train.progress	2	A Hype Train makes progress on the specified channel.
// Hype Train End	channel.hype_train.end	2	A Hype Train ends on the specified channel.
// Shield Mode Begin	channel.shield_mode.begin	1	Sends a notification when the broadcaster activates Shield Mode.
// Shield Mode End	channel.shield_mode.end	1	Sends a notification when the broadcaster deactivates Shield Mode.
// Shoutout Create	channel.shoutout.create	1	Sends a notification when the specified broadcaster sends a Shoutout.
// Shoutout Received	channel.shoutout.receive	1	Sends a notification when the specified broadcaster receives a Shoutout.
// Stream Online	stream.online	1	The specified broadcaster starts a stream.
// Stream Offline	stream.offline	1	The specified broadcaster stops a stream.
// User Authorization Grant	user.authorization.grant	1	A user’s authorization has been granted to your client id.
// User Authorization Revoke	user.authorization.revoke	1	A user’s authorization has been revoked for your client id.
// User Update	user.update	1	A user has updated their account.
// Whisper Received
// NEW	user.whisper.message	1	A user receives a whisper.
)