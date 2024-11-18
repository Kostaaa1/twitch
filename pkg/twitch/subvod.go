package twitch

import (
	"fmt"
	"strings"
	"time"
)

type SubVODResponse struct {
	Video struct {
		BroadcastType string    `json:"broadcastType"`
		CreatedAt     time.Time `json:"createdAt"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
		SeekPreviewsURL string `json:"seekPreviewsURL"`
	} `json:"video"`
}

func (api *API) SubVODData(vodID string) (SubVODResponse, error) {
	gqlPayload := `{
 	   "query": "query { video(id: \"%s\") { broadcastType, createdAt, seekPreviewsURL, owner { login } } }"
	}`
	body := strings.NewReader(fmt.Sprintf(gqlPayload, vodID))

	var subVodResponse struct {
		Data SubVODResponse `json:"data"`
	}
	if err := api.sendGqlLoadAndDecode(body, &subVodResponse); err != nil {
		return SubVODResponse{}, err
	}
	return subVodResponse.Data, nil
}
