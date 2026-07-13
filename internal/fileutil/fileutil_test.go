package fileutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstructPathname_NewUniqueFile(t *testing.T) {
	var f, f2 *os.File

	dir, file, ext := os.TempDir(), "video", "mp4"

	pathname, err := ConstructPathname(dir, file, ext)
	require.NoError(t, err)
	require.Equal(t, pathname, filepath.Join(dir, "video.mp4"))

	f, err = os.Create(pathname)
	require.NoError(t, err)
	require.Equal(t, f.Name(), filepath.Join(dir, "video.mp4"))

	pathname, err = ConstructPathname(dir, file, ext)
	require.NoError(t, err)
	require.Equal(t, pathname, filepath.Join(dir, "video (1).mp4"))

	f2, err = os.Create(pathname)
	require.NoError(t, err)
	require.Equal(t, f2.Name(), filepath.Join(dir, "video (1).mp4"))
}

func TestConstructPathname_Validate(t *testing.T) {
	dir := os.TempDir()

	tests := []struct {
		name           string
		dir, file, ext string
		wantErr        bool
	}{
		{name: "valid: proper inputs", dir: dir, file: "rand", ext: "mp4", wantErr: false},
		{name: "valid: extension with prefix", dir: dir, file: "rand", ext: ".mp4", wantErr: false},
		{name: "fail: missing dir", dir: "", file: "rand", ext: "mp4", wantErr: true},
		{name: "fail: missing file", dir: dir, file: "", ext: "mp4", wantErr: true},
		{name: "fail: missing ext", dir: dir, file: "rand", ext: "", wantErr: true},
	}

	for _, tc := range tests {
		pathname, err := ConstructPathname(tc.dir, tc.file, tc.ext)
		if tc.wantErr {
			require.Error(t, err)
			require.Empty(t, pathname)
		} else {
			require.NoError(t, err)
			require.NotEmpty(t, pathname)
		}
	}
}

func TestConstructPathname_DirDoesNotExist(t *testing.T) {
	pathname, err := ConstructPathname("./invalid/dir", "video", "mp4")
	require.Error(t, err)
	require.Empty(t, pathname)
}
