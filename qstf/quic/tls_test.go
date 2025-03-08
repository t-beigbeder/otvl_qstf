package quic

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigInsecureNoAlpn(t *testing.T) {
	tc, qc, err := GetConfig(&QuicOptions{TlsOptions: TlsOptions{InsecureSkipVerify: true}})
	require.NoError(t, err)
	require.NotNil(t, tc)
	require.NotNil(t, qc)
	port, cancel, err := netutils.RunTestServer("", func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetConfigInsecureNoAlpn: %s\n", cn.LocalAddr().String())
	}, netutils.GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	cnc, err := netutils.NewQuicConn("localhost:"+port, 0, tc, qc)
	require.NoError(t, err)
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
	port, cancel, err := netutils.RunTestServer("TestGetConfigInsecureAlpn", func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetConfigInsecureAlpn: %s\n", cn.LocalAddr().String())
	}, netutils.GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	cnc, err := netutils.NewQuicConn("localhost:"+port, 0, tc, qc)
	require.NoError(t, err)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
}

func TestGetConfigSecureAlpn(t *testing.T) {
	td := t.TempDir()
	err := netutils.NewCaCertFiles(
		filepath.Join(td, "cacert.pem"),
		filepath.Join(td, "cacert-key.pem"))
	require.NoError(t, err)
	err = netutils.NewCertFiles(nil,
		filepath.Join(td, "cacert.pem"),
		filepath.Join(td, "cacert-key.pem"),
		filepath.Join(td, "cert.pem"),
		filepath.Join(td, "cert-key.pem"),
	)
	require.NoError(t, err)

	tc, qc, err := GetConfig(&QuicOptions{
		TlsOptions: TlsOptions{
			CertFile:   filepath.Join(td, "cert.pem"),
			KeyFile:    filepath.Join(td, "cert-key.pem"),
			CACertFile: filepath.Join(td, "cacert.pem"),
		},
		Alpns: netutils.NextProtosFor("TestGetConfigSecureAlpn"),
	})
	require.NoError(t, err)
	require.NotNil(t, tc)
	require.NotNil(t, qc)
	port, cancel, err := netutils.RunTestServer("TestGetConfigSecureAlpn", func(ctx context.Context, cn quic.Connection, _ *slog.Logger) {
		fmt.Fprintf(os.Stderr, "TestGetConfigSecureAlpn: %s\n", cn.LocalAddr().String())
	}, netutils.GetLoggerFor("server"))
	require.NoError(t, err)
	defer cancel()
	cnc, err := netutils.NewQuicConn("localhost:"+port, 0, tc, qc)
	require.NoError(t, err)
	err = cnc.CloseWithError(0, "no issue")
	require.NoError(t, err)
}
