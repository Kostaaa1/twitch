package gql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Kostaaa1/twitch/internal/httputil"
)

const (
	gqlURL   = "https://gql.twitch.tv/gql"
	clientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"
)

type Client struct {
	http *http.Client
}

type GQLError struct {
	Message string `json:"message,omitempty"`
}

type GQLResponse[T any] struct {
	Data       T          `json:"data"`
	Errors     []GQLError `json:"errors,omitempty"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{http: httpClient}
}

// func (c *Client) collectUnitsForChannels(ctx context.Context, units []*downloader.Unit) error {
// 	for _, channel := range channels {
// 		if limitVideos > 0 {
// 			data, err := gqlClient.FilterableVideoTower_Videos(ctx, gql.FilterableVideoTowerArgs{
// 				Limit:           limitVideos,
// 				Channel:         channel,
// 				BroadcasterType: gql.VideoBroadcastType(filterVideosBy),
// 				VideoSort:       gql.VideoSort(sortVideosBy),
// 			})
// 			if err != nil {
// 				return err
// 			}

// 			fmt.Println("VIDEOS:", data)
// 			for _, vod := range data.User.Videos.Edges {
// 				downloader.NewUnit(
// 					vod.Node.ID,
// 				)
// 			}
// 		}

// 		if limitClips > 0 {
// 			data, err := gqlClient.ClipsCardsUser(ctx, gql.ClipsCardsUserArgs{
// 				Channel: channel,
// 				Limit:   limitClips,
// 				Filter:  gql.LastDay,
// 			})
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Println("CLIPS:", data)
// 		}
// 	}

// 	return nil
// }

func sendGqlLoadAndDecode[T any](
	ctx context.Context,
	httpClient *http.Client,
	dst *T,
	gqlLoad string,
	a ...any,
) error {
	var resp GQLResponse[T]

	var reader io.Reader
	if len(a) > 0 {
		reader = strings.NewReader(fmt.Sprintf(gqlLoad, a...))
	} else {
		reader = strings.NewReader(gqlLoad)
	}

	h := http.Header{}
	h.Set("Client-Id", clientID)
	h.Set("Content-Type", "application/json")

	if err := httputil.DoJSON(
		ctx,
		httpClient,
		gqlURL,
		http.MethodPost,
		reader,
		&resp,
		h,
	); err != nil {
		return err
	}

	if len(resp.Errors) > 0 {
		var err error
		for _, gqlerr := range resp.Errors {
			err = errors.Join(err, errors.New(gqlerr.Message))
		}
		return err
	}

	*dst = resp.Data

	return nil
}
