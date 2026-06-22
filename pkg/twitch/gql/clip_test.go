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

func TestVideoTitle(t *testing.T) {
	gql := New(http.DefaultClient)
	ctx := context.Background()
	title, err := gql.VideoTitle(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, title)
}

func TestStreamTitle(t *testing.T) {
	gql := New(http.DefaultClient)
	ctx := context.Background()
	title, err := gql.StreamTitle(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, title)
}
