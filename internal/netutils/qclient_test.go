package netutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestGetQuicConnBasic(t *testing.T) {
	port, cancel, err := RunTestServer("TestGetQuicConnBasic", func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetQuicConn: %s\n", cn.LocalAddr().String())
	}, GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	tc := &tls.Config{NextProtos: []string{"TestGetQuicConnBasic"}, InsecureSkipVerify: true}
	cnc, err := NewQuicConn("localhost:"+port, 0, tc, nil)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
}

func TestGetQuicConnNoAlpn(t *testing.T) {
	port, cancel, err := RunTestServer("", func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetQuicConn: %s\n", cn.LocalAddr().String())
	}, GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	tc := &tls.Config{InsecureSkipVerify: true}
	cnc, err := NewQuicConn("localhost:"+port, 0, tc, nil)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
}
