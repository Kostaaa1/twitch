package downloader

import (
	"context"
	"testing"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/stretchr/testify/require"
)

func TestLivestreamRecording(t *testing.T) {
	c := twitch.NewClient(nil)
	unit := NewUnit("yugi2x")
	dl := New(c, nil)

	ctx := context.Background()
	err := dl.Download(ctx, *unit)
	require.NoError(t, err)
}
