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
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunSync(t *testing.T) {
	port, cancel, err := RunTestServer()
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 5; i++ {
		var sOut string
		err = ac.RunSyncFunction("TestNewAppClientBasic", fmt.Sprintf("#%03d", i), &sOut)
		require.NoError(t, err)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunFunc(t *testing.T) {
	port, cancel, err := RunTestServer()
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	os, err := ac.AddIStream("")
	require.NoError(t, err)
	is, err := ac.GetOStream("")
	require.NoError(t, err)
	_, _ = is, os
	fd, err := ac.NewFunction("TestNewAppClientRunFunc", "")
	require.NoError(t, err)
	_ = fd
	cancel()
	time.Sleep(20 * time.Millisecond)
}
