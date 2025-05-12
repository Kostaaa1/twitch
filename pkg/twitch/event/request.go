package event

type Transport struct {
	Method             string `json:"method"`
	SessionID          string `json:"session_id"`
	ConduitID          string `json:"conduit_id"`
	WebhookCallbackURL string `json:"callback"`
	Secret             string `json:"secret"`
}

type RequestBody struct {
	Version   int32                  `json:"version"`
	Type      Type                   `json:"type"`
	Condition map[string]interface{} `json:"condition"`
	Transport Transport              `json:"transport"`
}

func WebsocketTransport(sessionID string) Transport {
	return Transport{
		Method:    "websocket",
		SessionID: sessionID,
	}
}

func ConduitTransport(conduitID string) Transport {
	return Transport{
		Method:    "conduit",
		ConduitID: conduitID,
	}
}

func WebhookTransport(callbackURL, secret string) Transport {
	return Transport{
		Method:             "webhook",
		WebhookCallbackURL: callbackURL,
		Secret:             secret,
	}
}
