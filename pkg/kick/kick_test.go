package kick

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVOD(t *testing.T) {
	dl := New()
	data, err := dl.MasterPlaylistURL("destiny", "019ff21c-0f28-77f4-901f-0f64c45a3cbb")
	require.NoError(t, err)
	b, err := json.MarshalIndent(data, "", " ")
	require.NoError(t, err)
	fmt.Println(string(b))
}
