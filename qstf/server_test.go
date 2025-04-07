package qstf

import (
	"context"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestNewServerHost(t *testing.T) {
	td := t.TempDir()
	port, cancel, err := RunQstfTestServerWithCtc(td,
		func(ctx context.Context, connection quic.Connection, logger *slog.Logger) {

		},
		nil)
	require.NoError(t, err)
	require.NotNil(t, port)
	time.Sleep(time.Millisecond * 10)
	cancel()
}

func TestSimpleUseCase(t *testing.T) {
	// StrSrc(args) -> Flow(client) -> QstSnk(func, args) -> (net)
	// (net) -> QstSrc(func,args) -> Flow(func) -> StdoutSnk

}

func TestBasicSideChannel(t *testing.T) {
	// Same as TestSimpleUseCase but args are passed on a side channel
}
