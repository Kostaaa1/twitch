package eventsub

type subStatus string

const (
	StatusEnabled                            subStatus = "enabled"
	StatusWebhookCallbackVerificationPending subStatus = "webhook_callback_verification_pending"
	StatusWebhookCallbackVerificationFailed  subStatus = "webhook_callback_verification_failed"
	StatusNotificationFailuresExceeded       subStatus = "notification_failures_exceeded"
	StatusAuthorizationRevoked               subStatus = "authorization_revoked"
	StatusModeratorRemoved                   subStatus = "moderator_removed"
	StatusUserRemoved                        subStatus = "user_removed"
	StatusChatUserBanned                     subStatus = "chat_user_banned"
	StatusVersionRemoved                     subStatus = "version_removed"
	StatusBetaMaintenance                    subStatus = "beta_maintenance"
	StatusWebsocketDisconnected              subStatus = "websocket_disconnected"
	StatusWebsocketFailedPingPong            subStatus = "websocket_failed_ping_pong"
	StatusWebsocketReceivedInboundTraffic    subStatus = "websocket_received_inbound_traffic"
	StatusWebsocketConnectionUnused          subStatus = "websocket_connection_unused"
	StatusWebsocketInternalError             subStatus = "websocket_internal_error"
	StatusWebsocketNetworkTimeout            subStatus = "websocket_network_timeout"
	StatusWebsocketNetworkError              subStatus = "websocket_network_error"
	StatusWebsocketFailedToReconnect         subStatus = "websocket_failed_to_reconnect"
)
