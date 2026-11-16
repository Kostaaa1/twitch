package gql

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const testChannel = "twitch"

func TestVideoTitle(t *testing.T) {
	gql := New(http.DefaultClient)
	ctx := context.Background()
	title, err := gql.VideoTitle(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, title)
}

func TestFilterableVideoTower_Videos(t *testing.T) {
	gql := New(http.DefaultClient)
	ctx := context.Background()

	types := []VideoBroadcastType{VideoTypeAll, VideoTypeArchive, VideoTypeHighlight, VideoTypeUpload}
	sorts := []VideoSort{SortTime, SortViews}

	for _, bt := range types {
		for _, sort := range sorts {
			t.Run(fmt.Sprintf("%s/%s", bt, sort), func(t *testing.T) {
				videos, err := gql.FilterableVideoTower_Videos(ctx, FilterableVideoTowerArgs{
					Limit:           100,
					Channel:         testChannel,
					BroadcasterType: bt,
					VideoSort:       sort,
				})
				require.NoError(t, err)
				require.NotNil(t, videos)
				t.Log(len(videos.User.Videos.Edges))
			})
		}
	}
}
