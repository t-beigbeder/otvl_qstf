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
	logger *slog.Logger,
) (*ServerHost, string, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	const testHost = "localhost"
	var (
		err        error
		cfs        map[string]string
		stc        *tls.Config
		sqc        *quic.Config
		listener   *quic.Listener
		host, port string
	)
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	cfs, err = netutils.NewTestCerts(testDir, []string{testHost}, false)
	if err != nil {
		return nil, "", nil, err
	}
	stc, sqc, err = quicutils.GetConfig(&quicutils.QuicOptions{
		IsServer:   true,
		TlsOptions: quicutils.TlsOptions{CertFile: cfs["svc"], KeyFile: cfs["svk"]},
		Alpns:      netutils.NextProtosFor(QstfAlpn),
	})
	if err != nil {
		return nil, "", nil, err
	}
	listener, host, port, err = netutils.GetQuicListener(":0", stc, sqc)
	if err != nil {
		return nil, "", nil, err
	}
	if logger == nil {
		logger = common.GetLoggerFor("test-server")
	}
	logger.Info("RunQstfTestServerWithCtc: listening", "host", host, "port", port)
	sh := NewServerHost(ctx, logger, listener, "test-server-id")

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
	return sh, port, cancel, nil
}
