package gql

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamTitle(t *testing.T) {
	gql := New(http.DefaultClient)
	ctx := context.Background()
	title, err := gql.StreamTitle(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, title)
}
