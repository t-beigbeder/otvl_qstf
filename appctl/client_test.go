package appctl

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewAppClientBasic(t *testing.T) {
	port, cancel, err := RunTestServer()
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 5; i++ {
		var sOut string
		err = ac.RunSyncFunction("TestNewAppClientBasic", fmt.Sprintf("#%03d", i), &sOut)
		require.NoError(t, err)
	}
	cancel()
	time.Sleep(100 * time.Millisecond)
}
