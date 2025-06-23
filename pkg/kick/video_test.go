package kick_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDownloadVOD(t *testing.T) {
	t.Parallel()

	c := kick.NewClient()
	channel := "asmongold"

	videos, err := c.GetVideos(channel)
	require.NoError(t, err)
	require.NotNil(t, videos)

	newFilePath := fmt.Sprintf("/mnt/c/Users/Kosta/Downloads/Clips/%s.mp4", uuid.New().String())
	newFile, err := os.Create(newFilePath)
	require.NoError(t, err)

	unit := kick.Unit{
		VodID:   videos[0].Video.UUID,
		Channel: channel,
		Writer:  newFile,
	}

	err = c.DownloadVOD(context.Background(), unit)
	require.NoError(t, err)
}

func TestGetVideos(t *testing.T) {
	t.Parallel()

	c := kick.NewClient()
	channel := "asmongold"

	videos, err := c.GetVideos(channel)
	require.Nil(t, err)
	require.NotNil(t, videos)
}

func TestGetVideo(t *testing.T) {
	t.Parallel()

	c := kick.NewClient()

	videoID := "c57eff06-46de-4a26-a791-590d6a6d8967"
	video, err := c.GetVideoByID(videoID)
	require.NoError(t, err)
	require.NotNil(t, video)
}
