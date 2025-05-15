package event

type Transport struct {
	Method             string `json:"method"`
	SessionID          string `json:"session_id"`
	ConduitID          string `json:"conduit_id"`
	WebhookCallbackURL string `json:"callback"`
	Secret             string `json:"secret"`
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
