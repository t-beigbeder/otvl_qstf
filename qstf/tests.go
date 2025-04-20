package qstf

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/qstf/quicutils"
	"log/slog"
	"time"
)

const testHost = "localhost"

func RunQstfTestServer(
	testDir string,
	configureServer func(ServerHost) error,
) (ServerHost, string, context.CancelFunc, map[string]string, error) {
	ctx, cCancel := context.WithCancel(context.Background())
	var (
		cfs        map[string]string
		err        error
		stc        *tls.Config
		sqc        *quic.Config
		listener   *quic.Listener
		host, port string
	)
	defer func() {
		if err != nil {
			cCancel()
		}
	}()
	cfs, err = netutils.NewTestCerts(testDir, []string{testHost}, true)
	if err != nil {
		return nil, "", nil, nil, err
	}
	stc, sqc, err = quicutils.GetConfig(&quicutils.QuicOptions{
		IsServer:   true,
		TlsOptions: quicutils.TlsOptions{CertFile: cfs["svc"], KeyFile: cfs["svk"]},
		Alpns:      netutils.NextProtosFor(QstfAlpn),
	})

	listener, host, port, err = netutils.GetQuicListener(":0", stc, sqc)
	if err != nil {
		return nil, "", nil, nil, err
	}
	logger := common.GetLoggerFor("test-server")
	logger.Info("RunQstfTestServer: listening", "host", host, "port", port)
	sh := NewServerHost(ctx, logger, listener, "test-server-id")

	ready := err != nil
	var lastErr error
	ctc := &tls.Config{NextProtos: stc.NextProtos, InsecureSkipVerify: true}
	for cc := 0; cc < 3 && !ready; cc++ {
		timeout := time.Duration(20*(cc+1)) * time.Millisecond

		ccn, ierr := netutils.NewQuicConn(fmt.Sprintf("%s:%s", testHost, port), timeout, ctc, nil)
		if ierr == nil {
			ierr2 := ccn.CloseWithError(0, "")
			_ = ierr2
			ready = true
		}
		lastErr = ierr
	}
	if !ready && err == nil {
		err = fmt.Errorf("runQstfTestServerWithCtc: failed to connect to %s:%s err %v", host, port, lastErr)
	}
	if configureServer != nil {
		if err = configureServer(sh); err != nil {
			return nil, "", nil, nil, err
		}
	}
	cancel := func() {
		cCancel()
		sh.Shutdown()
	}
	return sh, port, cancel, cfs, nil
}

func RunQstfTestClientServer(
	testDir string,
	configureServer func(ServerHost) error,
	clientDoer func(ClientHost, Connector, *slog.Logger) error,
) (context.CancelFunc, error) {
	sh, port, cancel, cfs, err := RunQstfTestServer(testDir, configureServer)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	logger := common.GetLoggerFor("test-client")
	ch := NewClientHost(logger, "TestClientBasicHostId")
	qo := &quicutils.QuicOptions{
		TlsOptions: quicutils.TlsOptions{CACertFile: cfs["cac"]},
		Alpns:      netutils.NextProtosFor(QstfAlpn),
	}
	cnt, err := NewConnector(testHost+":"+port, sh.HostId(), qo, 0, logger)
	if err != nil {
		return nil, err
	}
	stc, err := ch.OpenStream(cnt, "", "")
	if err != nil {
		return nil, err
	}

	err = clientDoer(ch, cnt, logger)
	if err != nil {
		return nil, err
	}
	err = stc.Close()
	if err != nil {
		return nil, err
	}
	return cancel, nil
}
