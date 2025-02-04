package appctl

import (
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"testing"
	"time"
)

func TestAppServerConnectionHandlerBasic(t *testing.T) {
	_, cancel, err := netutils.RunTestServer(QstfAlpn, quicAppServerConnectionHandler)
	require.NoError(t, err)
	cancel()
	time.Sleep(100 * time.Millisecond)
}
