package qstf

import (
	"context"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/qstf/quicutils"
	"log/slog"
	"testing"
	"time"
)

func TestNewServerHostStarter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	td := t.TempDir()
	cfs, err := netutils.NewTestCerts(td, []string{"localhost"}, false)
	require.NoError(t, err)
	stc, sqc, err := quicutils.GetConfig(&quicutils.QuicOptions{
		IsServer:   true,
		TlsOptions: quicutils.TlsOptions{CertFile: cfs["svc"], KeyFile: cfs["svk"]},
		Alpns:      netutils.NextProtosFor("TestNewServerHost"),
	})
	require.NoError(t, err)
	require.NotNil(t, stc)
	require.NotNil(t, sqc)
	lst, host, port, err := netutils.GetQuicListener(":0", stc, sqc)
	require.NoError(t, err)
	require.NotNil(t, lst)
	require.NotEmpty(t, host)
	require.NotEmpty(t, port)
	hid, err := uuid.NewV7()
	require.NoError(t, err)
	NewServerHost(ctx, common.GetLoggerFor("server"), lst, hid.String())
	time.Sleep(time.Millisecond * 10)
	cancel()
}

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
