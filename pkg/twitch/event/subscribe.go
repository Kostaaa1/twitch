package event

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch"
)

type Subscription struct {
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
	Data         []Subscription         `json:"data"`
	Total        int                    `json:"total"`
	TotalCost    int                    `json:"total_cost"`
	MaxTotalCost int                    `json:"max_total_cost"`
	Pagination   map[string]interface{} `json:"pagination"`
}

func (sub *EventSub) Subscribe(body RequestBody) (*SubscriptionResponse, error) {
	url := "https://api.twitch.tv/helix/eventsub/subscriptions"
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	data, err := twitch.HelixRequest[SubscriptionResponse](sub.tw, url, http.MethodPost, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	sub.TotalCost = data.TotalCost
	sub.Total = data.Total
	sub.MaxTotalCost = data.MaxTotalCost

	sub.Subscriptions = append(sub.Subscriptions, data.Data[0])
	return data, nil
}

func (sub *EventSub) GetSubscriptions() (*SubscriptionResponse, error) {
	url := "https://api.twitch.tv/helix/eventsub/subscriptions"
	data, err := twitch.HelixRequest[SubscriptionResponse](sub.tw, url, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (sub *EventSub) RemoveSubscriptionByID(id string) {
	newSubs := sub.Subscriptions[:0]
	for _, s := range sub.Subscriptions {
		if s.ID != id {
			newSubs = append(newSubs, s)
		}
	}
	sub.Subscriptions = newSubs
}

func (sub *EventSub) DeleteSubscription(ctx context.Context, subId string) error {
	url := "https://api.twitch.tv/helix/eventsub/subscriptions?id=" + subId
	_, err := twitch.HelixRequestWithCtx[interface{}](ctx, sub.tw, url, http.MethodDelete, nil)
	if err != nil {
		return err
	}
	sub.RemoveSubscriptionByID(subId)
	return nil
}

func (sub *EventSub) DeleteAllSubscriptions(ctx context.Context) error {
	subCopy := make([]Subscription, len(sub.Subscriptions))
	copy(subCopy, sub.Subscriptions)
	for _, data := range subCopy {
		if err := sub.DeleteSubscription(ctx, data.ID); err != nil {
			return err
		}
	}
	return nil
}
