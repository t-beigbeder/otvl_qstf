package netutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"log/slog"
	"os"
	"time"
)

func RunTestServer(
	alpn string,
	cert *tls.Certificate,
	doer func(ctx context.Context, connection quic.Connection, logger *slog.Logger),
	logger *slog.Logger,
) (string, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var (
		host, port string
		err        error
	)
	go func() {
		logger := logger
		listener, ihost, iport, ierr := GetQuicListener(":0", cert, alpn, logger)
		logger.Info("RunTestServer: listening", "host", ihost, "port", iport)
		if ierr != nil {
			err = ierr
			return
		}
		host, port = ihost, iport
		var checked bool
		for {
			cnc, ierr := listener.Accept(ctx)
			if ierr != nil {
				logger.Error("RunTestServer: accept error", "host", host, "port", iport, "err", ierr)
				return
			}
			logger.Info("RunTestServer: accepted connection", "host", host, "port", iport, "remoteAddr", cnc.RemoteAddr().String())
			if checked {
				doer(ctx, cnc, logger)
			} else {
				checked = true
			}
		}
	}()
	ready := err != nil
	var lastErr error
	for cc := 0; cc < 3 && !ready; cc++ {
		timeout := time.Duration(10*(cc+1)) * time.Millisecond
		tc := &tls.Config{NextProtos: NextProtosFor(alpn), InsecureSkipVerify: true}
		ccn, ierr := NewQuicConn(fmt.Sprintf("%s:%s", "localhost", port), timeout, tc, nil)
		if ierr == nil {
			ccn.CloseWithError(0, "")
			ready = true
		}
		lastErr = ierr
	}
	if !ready && err == nil {
		err = fmt.Errorf("RunTestServer: failed to connect to %s:%s err %v", host, port, lastErr)
	}
	if err != nil {
		cancel()
		return "", nil, err
	}
	return port, cancel, nil
}

func GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func GetLoggerFor(app string) *slog.Logger {
	return GetLogger().With("app", app)
}
