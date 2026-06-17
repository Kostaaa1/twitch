package eventsub

type transportMethod string

const (
	Webhook   transportMethod = "webhook"
	Websocket transportMethod = "websocket"
	Conduit   transportMethod = "conduit"
)

type webhookTransport struct {
	Method   transportMethod `json:"method"`
	Callback string          `json:"callback"`
	Secret   string          `json:"secret"`
}

type websocketTransport struct {
	Method         transportMethod `json:"method"`
	SessionID      string          `json:"session_id"`
	ConnectedAt    string          `json:"connected_at"`
	DisconnectedAt string          `json:"disconnected_at"`
}

type transport struct {
	Method transportMethod `json:"method"`
	// webhook
	Callback string `json:"callback,omitempty"`
	Secret   string `json:"secret,omitempty"`
	// websocket
	SessionID string `json:"session_id,omitempty"`
	// ConnectedAt    string `json:"connected_at,omitempty"`
	// DisconnectedAt string `json:"disconnected_at"`
}
