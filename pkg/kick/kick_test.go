package kick

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"strings"
// 	"testing"

// 	"github.com/Kostaaa1/twitch/pkg/m3u8"
// 	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
// 	"github.com/stretchr/testify/require"
// )

// func TestKick_GetMasterPlaylist(t *testing.T) {
// 	t.Parallel()

// 	client := NewClient()

// 	// u := "https://kick.com/asmongold/videos/0897d891-f6db-4b71-be32-9b88b609974a"
// 	u := "https://kick.com/cutegirlasmr/videos/cc0decca-67a0-4e78-8476-04bc32c2c9db"

// 	masterURL, err := client.MasterPlaylistURL(u)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, masterURL)

// 	fmt.Println("MASTER:", masterURL)

// 	var buf bytes.Buffer
// 	unit := Unit{
// 		URL:     "https://kick.com/cutegirlasmr/videos/cc0decca-67a0-4e78-8476-04bc32c2c9db",
// 		W:       &buf,
// 		Quality: downloader.Quality720p60,
// 	}

// 	basePath := strings.TrimSuffix(masterURL, "master.m3u8")
// 	playlistURL := basePath + unit.Quality.String() + "/playlist.m3u8"

// 	fmt.Println("PLAYLIST:", playlistURL)

// 	original := unit.Quality

// 	for {
// 		res, err := client.client.Get(playlistURL)
// 		defer res.Body.Close()
// 		require.NoError(t, err)

// 		if res.StatusCode == http.StatusForbidden {
// 			if unit.Quality == downloader.QualityWorst && original < downloader.Quality1080p60 {
// 				unit.Quality.Upgrade()
// 				continue
// 			}
// 			unit.Quality.Downgrade()
// 			continue
// 		}
// 		break
// 	}

// 	// require.NoError(t, err)

// 	playlist := m3u8.ParseMediaPlaylist(res.Body)
// 	playlist.TruncateSegments(unit.Start, unit.End)

// 	fmt.Println("SEGMENTS:", playlist.Segments)
// }

// func TestKick_DownloadVideo(t *testing.T) {
// 	client := NewClient()

// 	var buf bytes.Buffer

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
// 	defer cancel()
// 	err := client.Download(ctx, testUnit)
// 	require.NoError(t, err)
// }
