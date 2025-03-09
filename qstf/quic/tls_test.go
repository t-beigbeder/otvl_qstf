package quic

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestGetConfigInsecureNoAlpn(t *testing.T) {
	tc, qc, err := GetConfig(&QuicOptions{TlsOptions: TlsOptions{InsecureSkipVerify: true}})
	require.NoError(t, err)
	require.NotNil(t, tc)
	require.NotNil(t, qc)
	cert, err := netutils.SelfSigned("localhost")
	require.NoError(t, err)
	var flag bool
	port, cancel, err := netutils.RunQuicTestServerFor("", cert, func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetConfigInsecureNoAlpn: %s\n", cn.LocalAddr().String())
		flag = true
	}, netutils.GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	cnc, err := netutils.NewQuicConn("localhost:"+port, 0, tc, qc)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	require.True(t, flag)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
}

func TestGetConfigInsecureAlpn(t *testing.T) {
	tc, qc, err := GetConfig(&QuicOptions{
		TlsOptions: TlsOptions{InsecureSkipVerify: true},
		Alpns:      netutils.NextProtosFor("TestGetConfigInsecureAlpn")})
	require.NoError(t, err)
	require.NotNil(t, tc)
	require.NotNil(t, qc)
	cert, err := netutils.SelfSigned("localhost")
	require.NoError(t, err)
	var flag bool
	port, cancel, err := netutils.RunQuicTestServerFor("TestGetConfigInsecureAlpn", cert, func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetConfigInsecureAlpn: %s\n", cn.LocalAddr().String())
		flag = true
	}, netutils.GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	cnc, err := netutils.NewQuicConn("localhost:"+port, 0, tc, qc)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	require.True(t, flag)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
}

func TestGetConfigSecureAlpn(t *testing.T) {
	td := t.TempDir()
	cfs, err := netutils.NewTestCerts(td, []string{"localhost"})
	require.NoError(t, err)
	stc, sqc, err := GetConfig(&QuicOptions{
		IsServer:   true,
		TlsOptions: TlsOptions{CertFile: cfs["svc"], KeyFile: cfs["svk"]},
		Alpns:      netutils.NextProtosFor("TestGetConfigSecureAlpn"),
	})
	require.NotNil(t, stc)
	require.NotNil(t, sqc)
	var flag bool
	port, cancel, err := netutils.RunQuicTestServer(stc, sqc, func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetConfigSecureAlpn: %s\n", cn.LocalAddr().String())
		flag = true
	}, netutils.GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()

	ctc, cqc, err := GetConfig(&QuicOptions{
		TlsOptions: TlsOptions{CACertFile: cfs["cac"]},
		Alpns:      netutils.NextProtosFor("TestGetConfigSecureAlpn"),
	})
	require.NoError(t, err)
	cnc, err := netutils.NewQuicConn("localhost:"+port, 0, ctc, cqc)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	require.True(t, flag)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
}

func TestGetConfigClientServerAlpn(t *testing.T) {
	td := t.TempDir()
	cfs, err := netutils.NewTestCerts(td, []string{"localhost"})
	require.NoError(t, err)
	stc, sqc, err := GetConfig(&QuicOptions{
		IsServer:   true,
		TlsOptions: TlsOptions{CertFile: cfs["svc"], KeyFile: cfs["svk"], ClientAuth: true},
		Alpns:      netutils.NextProtosFor("TestGetConfigClientServerAlpn"),
	})
	require.NotNil(t, stc)
	require.NotNil(t, sqc)
	var flag bool
	port, cancel, err := netutils.RunQuicTestServer(stc, sqc, func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetConfigClientServerAlpn: %s\n", cn.LocalAddr().String())
		flag = true
	}, netutils.GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()

	ctc, cqc, err := GetConfig(&QuicOptions{
		TlsOptions: TlsOptions{CertFile: cfs["svc"], KeyFile: cfs["svk"], CACertFile: cfs["cac"]},
		Alpns:      netutils.NextProtosFor("TestGetConfigClientServerAlpn"),
	})
	require.NoError(t, err)
	cnc, err := netutils.NewQuicConn("localhost:"+port, 0, ctc, cqc)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	require.True(t, flag)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
}
