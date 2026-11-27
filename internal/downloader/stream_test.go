package downloader

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloader_Stream(t *testing.T) {}

func TestNewSegments(t *testing.T) {
	tests := []struct {
		name     string
		last     string
		playlist []string
		send     []string
		newLast  string
	}{
		{name: "first poll", last: "", playlist: []string{"A", "B", "C"}, send: []string{"A", "B", "C"}, newLast: "C"},
		{name: "first poll is empty", last: "", playlist: nil, send: nil, newLast: ""},
		{name: "one new", last: "C", playlist: []string{"B", "C", "D"}, send: []string{"D"}, newLast: "D"},
		{name: "several new", last: "C", playlist: []string{"B", "C", "D", "E"}, send: []string{"D", "E"}, newLast: "E"},
		{name: "nothing new", last: "C", playlist: []string{"A", "B", "C"}, send: nil, newLast: "C"},
		{name: "empty playlist keeps position", last: "C", playlist: nil, send: nil, newLast: "C"},
		{name: "last dropped out of the window", last: "C", playlist: []string{"F", "G", "H"}, send: []string{"F", "G", "H"}, newLast: "H"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			send, newLast := newSegments(tc.playlist, tc.last)
			// order matters, nil and empty are both "send nothing"
			require.Len(t, send, len(tc.send))
			for i := range tc.send {
				require.Equal(t, tc.send[i], send[i])
			}
			require.Equal(t, tc.newLast, newLast)
		})
	}
}

func TestNewSegments_AcrossPolls(t *testing.T) {
	polls := [][]string{
		{"A", "B", "C"},
		{"B", "C", "D"},
		{"B", "C", "D"}, // nothing new
		{},              // failed or only ads
		{"C", "D", "E"},
		{"H", "I"}, // missed F and G
	}
	last := ""
	got := make([]string, 0)
	for id, poll := range polls {
		var send []string
		send, last = newSegments(poll, last)
		fmt.Printf("Got: id=%d send=%s last=%s\n", id, send, last)
		got = append(got, send...)
	}
	require.Equal(t, []string{"A", "B", "C", "D", "E", "H", "I"}, got)
}
