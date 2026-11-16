package gql

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClipTitle(t *testing.T) {
	gql := New(http.DefaultClient)
	ctx := context.Background()
	title, err := gql.ClipTitle(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, title)
}

func TestClipsCardsUser(t *testing.T) {
	gql := New(http.DefaultClient)
	ctx := context.Background()

	filters := []CriteriaFilter{AllTime, LastDay, LastWeek, LastMonth}

	for _, filter := range filters {
		t.Run(string(filter), func(t *testing.T) {
			data, err := gql.ClipsCardsUser(ctx, ClipsCardsUserArgs{
				Channel: testChannel,
				Filter:  filter,
				Limit:   10,
			})
			require.NoError(t, err)
			require.NotNil(t, data)
		})
	}
}
