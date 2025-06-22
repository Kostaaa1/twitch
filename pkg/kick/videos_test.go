package kick_test

import (
	"testing"

	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/stretchr/testify/require"
)

var (
	channel = "soyununicornio1"
)

func TestGetVideos(t *testing.T) {
	t.Parallel()

	c := kick.NewClient()

	videos, err := c.GetVideos(channel)
	require.NoError(t, err)
	require.NotNil(t, videos)
}

func TestGetVideoByUUID(t *testing.T) {
	t.Parallel()

	c := kick.NewClient()

	videos, err := c.GetVideos(channel)
	require.NoError(t, err)
	require.NotNil(t, videos)
	require.Greater(t, len(videos), 0)

	uuid := videos[0].Video.UUID
	video, err := c.GetVideoByUUID(uuid)
	require.NoError(t, err)
	require.NotNil(t, video)
}

func TestMasterM3u8(t *testing.T) {
	c := kick.NewClient()

	videos, err := c.GetVideos(channel)
	require.NoError(t, err)
	require.NotNil(t, videos)
	require.Greater(t, len(videos), 0)

	uuid := videos[0].Video.UUID

	masterURL, err := c.GetMasterPlaylistURL(channel, uuid)
	require.NoError(t, err)
	require.NotEmpty(t, masterURL)

	_, err = c.FetchMediaPlaylist(masterURL, "1080")
	require.NoError(t, err)

	// parsedPlaylist := m3u8.Master([]byte(master))
	// t.Log(parsedPlaylist)
}
