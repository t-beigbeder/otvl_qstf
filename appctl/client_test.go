package appctl

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"testing"
	"time"
)

func TestNewAppClientBasic(t *testing.T) {
	port, cancel, err := netutils.RunTestServer(QstfAlpn, quicAppServerConnectionHandler)
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port)
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(100 * time.Millisecond)
	fc, err := ac.RunFunction("Go", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, fc)
	cancel()
	time.Sleep(100 * time.Millisecond)
}
