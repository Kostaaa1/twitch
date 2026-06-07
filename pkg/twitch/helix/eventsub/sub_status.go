package eventsub

type SubStatus string

const (
	StatusEnabled                            SubStatus = "enabled"
	StatusWebhookCallbackVerificationPending SubStatus = "webhook_callback_verification_pending"
	StatusWebhookCallbackVerificationFailed  SubStatus = "webhook_callback_verification_failed"
	StatusNotificationFailuresExceeded       SubStatus = "notification_failures_exceeded"
	StatusAuthorizationRevoked               SubStatus = "authorization_revoked"
	StatusModeratorRemoved                   SubStatus = "moderator_removed"
	StatusUserRemoved                        SubStatus = "user_removed"
	StatusChatUserBanned                     SubStatus = "chat_user_banned"
	StatusVersionRemoved                     SubStatus = "version_removed"
	StatusBetaMaintenance                    SubStatus = "beta_maintenance"
	StatusWebsocketDisconnected              SubStatus = "websocket_disconnected"
	StatusWebsocketFailedPingPong            SubStatus = "websocket_failed_ping_pong"
	StatusWebsocketReceivedInboundTraffic    SubStatus = "websocket_received_inbound_traffic"
	StatusWebsocketConnectionUnused          SubStatus = "websocket_connection_unused"
	StatusWebsocketInternalError             SubStatus = "websocket_internal_error"
	StatusWebsocketNetworkTimeout            SubStatus = "websocket_network_timeout"
	StatusWebsocketNetworkError              SubStatus = "websocket_network_error"
	StatusWebsocketFailedToReconnect         SubStatus = "websocket_failed_to_reconnect"
)
