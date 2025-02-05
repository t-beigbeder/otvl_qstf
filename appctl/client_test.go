package appctl

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewAppClientBasic(t *testing.T) {
	port, cancel, err := RunTestServer()
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port)
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(100 * time.Millisecond)
	var sOut string
	err = ac.RunSyncFunction("Go", "TestNewAppClientBasic", &sOut)
	require.NoError(t, err)
	cancel()
	time.Sleep(100 * time.Millisecond)
}
