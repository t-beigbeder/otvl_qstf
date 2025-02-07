package appctl

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAppServerConnectionHandlerBasic(t *testing.T) {
	_, cancel, err := RunTestServer(nil)
	require.NoError(t, err)
	cancel()
	time.Sleep(100 * time.Millisecond)
}
