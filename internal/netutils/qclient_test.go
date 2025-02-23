package netutils

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestGetQuicConn(t *testing.T) {
	port, cancel, err := RunTestServer("TestGetQuicConn", func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetQuicConn: %s\n", cn.LocalAddr().String())
	}, GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	cnc, err := GetQuicConn("localhost:"+port, "TestGetQuicConn", 0)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
}
