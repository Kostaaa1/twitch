package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SubscriptionMessage struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Type      string                 `json:"type"`
	Version   string                 `json:"version"`
	Condition map[string]interface{} `json:"condition"`
	CreatedAt time.Time              `json:"created_at"`
	Transport Transport              `json:"transport"`
	Cost      int                    `json:"cost"`
}

type SubscriptionResponse struct {
	Data         []SubscriptionMessage  `json:"data"`
	Total        int                    `json:"total"`
	TotalCost    int                    `json:"total_cost"`
	MaxTotalCost int                    `json:"max_total_cost"`
	Pagination   map[string]interface{} `json:"pagination"`
}

var (
	subscriptionsURL = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

func (sub *EventSubClient) Subscribe(body RequestBody) (*SubscriptionResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var data SubscriptionResponse
	if err := sub.tw.HelixRequest(subscriptionsURL, http.MethodPost, bytes.NewBuffer(b), &data); err != nil {
		return nil, err
	}

	sub.Subscriptions = append(sub.Subscriptions, data.Data[0])
	return &data, nil
}

func (sub *EventSubClient) GetSubscriptions() (*SubscriptionResponse, error) {
	var data SubscriptionResponse
	if err := sub.tw.HelixRequest(subscriptionsURL, http.MethodGet, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (sub *EventSubClient) RemoveSubscriptionByID(id string) {
	newSubs := sub.Subscriptions[:0]
	for _, s := range sub.Subscriptions {
		if s.ID != id {
			newSubs = append(newSubs, s)
		}
	}
	sub.Subscriptions = newSubs
}

func (sub *EventSubClient) Unsubscribe(subId string) error {
	url := fmt.Sprintf("%s?id=%s", subscriptionsURL, subId)
	if err := sub.tw.HelixRequest(url, http.MethodDelete, nil, nil); err != nil {
		return err
	}
	sub.RemoveSubscriptionByID(subId)
	return nil
}

func (sub *EventSubClient) UnsubscribeToAll() error {
	subCopy := make([]SubscriptionMessage, len(sub.Subscriptions))
	copy(subCopy, sub.Subscriptions)
	for _, data := range subCopy {
		if err := sub.Unsubscribe(data.ID); err != nil {
			return err
		}
	}
	return nil
}
