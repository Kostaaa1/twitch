package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type SubscriptionResponse struct {
	Data         []Subscription `json:"data"`
	Total        int            `json:"total"`
	TotalCost    int            `json:"total_cost"`
	MaxTotalCost int            `json:"max_total_cost"`
}

func (sub *EventSub) Subscribe(body RequestBody) (*SubscriptionResponse, error) {
	fmt.Println("Subscribing to the event: ", body)
	url := "https://api.twitch.tv/helix/eventsub/subscriptions"

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	data, err := twitch.HelixRequest[SubscriptionResponse](sub.tw, url, http.MethodPost, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	return data, nil
}
