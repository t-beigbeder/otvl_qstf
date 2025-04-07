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

func RunQstfTestServerWithCtc(
	testDir string,
	doer func(ctx context.Context, connection quic.Connection, logger *slog.Logger),
	logger *slog.Logger,
) (string, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	const testHost = "localhost"
	var (
		err        error
		cfs        map[string]string
		stc        *tls.Config
		sqc        *quic.Config
		host, port string
	)
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	cfs, err = netutils.NewTestCerts(testDir, []string{testHost}, false)
	if err != nil {
		return "", nil, err
	}
	stc, sqc, err = quicutils.GetConfig(&quicutils.QuicOptions{
		IsServer:   true,
		TlsOptions: quicutils.TlsOptions{CertFile: cfs["svc"], KeyFile: cfs["svk"]},
		Alpns:      netutils.NextProtosFor(QstfAlpn),
	})
	if err != nil {
		return "", nil, err
	}
	go func() {
		logger := logger
		if logger == nil {
			logger = common.GetLoggerFor("test-server")
		}
		listener, ihost, iport, ierr := netutils.GetQuicListener(":0", stc, sqc)
		logger.Info("RunQstfTestServerWithCtc: listening", "host", ihost, "port", iport)
		if ierr != nil {
			err = ierr
			return
		}
		host, port = ihost, iport
		var checked bool
		for {
			cnc, ierr := listener.Accept(ctx)
			if ierr != nil {
				logger.Error("RunQstfTestServerWithCtc: accept error", "host", host, "port", iport, "err", ierr)
				return
			}
			if checked {
				logger.Info("RunQstfTestServerWithCtc: accepted connection", "host", host, "port", iport, "remoteAddr", cnc.RemoteAddr().String(), "checked", checked)
				doer(ctx, cnc, logger)
			} else {
				logger.Info("RunQstfTestServerWithCtc: accepted check test connection", "host", host, "port", iport, "remoteAddr", cnc.RemoteAddr().String(), "checked", checked)
				checked = true
			}
		}
	}()
	ready := err != nil
	var lastErr error
	time.Sleep(10 * time.Millisecond)
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
		err = fmt.Errorf("RunQstfTestServerWithCtc: failed to connect to %s:%s err %v", host, port, lastErr)
	}
	return port, cancel, nil
}
