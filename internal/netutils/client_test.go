package netutils

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestGetQuicConn(t *testing.T) {
	port, cancel, err := RunTestServer("TestGetQuicConn", func(ctx context.Context, connection quic.Connection) {
		fmt.Fprintf(os.Stderr, "TestGetQuicConn: %v\n", connection)
	})
	require.NoError(t, err)
	defer cancel()
	cnc, err := GetQuicConn("localhost:"+port, "TestGetQuicConn")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
}
