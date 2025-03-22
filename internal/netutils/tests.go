package netutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func GetQuicConfigFor(cert *tls.Certificate, alpn string, logger *slog.Logger) (*tls.Config, *quic.Config) {
	qc := quic.Config{Tracer: qlog.DefaultConnectionTracer}
	tc := tls.Config{
		Certificates: []tls.Certificate{*cert},
		NextProtos:   NextProtosFor(alpn),
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			if logger != nil {
				logger.Info("connection", "ServerName", info.ServerName, "SupportedProtos", info.SupportedProtos)
			}
			return nil, nil
		},
	}
	return &tc, &qc
}

func RunQuicTestServerWithCtc(
	stc *tls.Config,
	ctc *tls.Config,
	qc *quic.Config,
	doer func(ctx context.Context, connection quic.Connection, logger *slog.Logger),
	logger *slog.Logger,
) (string, context.CancelFunc, error) {
	if ctc == nil {
		ctc = &tls.Config{NextProtos: stc.NextProtos, InsecureSkipVerify: true}
	}
	ctx, cancel := context.WithCancel(context.Background())
	var (
		host, port string
		err        error
	)
	go func() {
		logger := logger
		listener, ihost, iport, ierr := GetQuicListener(":0", stc, qc)
		logger.Info("RunQuicTestServer: listening", "host", ihost, "port", iport)
		if ierr != nil {
			err = ierr
			return
		}
		host, port = ihost, iport
		var checked bool
		for {
			cnc, ierr := listener.Accept(ctx)
			if ierr != nil {
				logger.Error("RunQuicTestServer: accept error", "host", host, "port", iport, "err", ierr)
				return
			}
			logger.Info("RunQuicTestServer: accepted connection", "host", host, "port", iport, "remoteAddr", cnc.RemoteAddr().String(), "checked", checked)
			if checked {
				doer(ctx, cnc, logger)
			} else {
				checked = true
			}
		}
	}()
	ready := err != nil
	var lastErr error
	time.Sleep(10 * time.Millisecond)
	for cc := 0; cc < 3 && !ready; cc++ {
		timeout := time.Duration(20*(cc+1)) * time.Millisecond

		ccn, ierr := NewQuicConn(fmt.Sprintf("%s:%s", "localhost", port), timeout, ctc, nil)
		if ierr == nil {
			ierr2 := ccn.CloseWithError(0, "")
			_ = ierr2
			ready = true
		}
		lastErr = ierr
	}
	if !ready && err == nil {
		err = fmt.Errorf("RunQuicTestServer: failed to connect to %s:%s err %v", host, port, lastErr)
	}
	if err != nil {
		cancel()
		return "", nil, err
	}
	return port, cancel, nil
}

func RunQuicTestServer(
	tc *tls.Config,
	qc *quic.Config,
	doer func(ctx context.Context, connection quic.Connection, logger *slog.Logger),
	logger *slog.Logger,
) (string, context.CancelFunc, error) {
	return RunQuicTestServerWithCtc(tc, nil, qc, doer, logger)
}

func RunQuicTestServerFor(
	alpn string,
	cert *tls.Certificate,
	doer func(ctx context.Context, connection quic.Connection, logger *slog.Logger),
	logger *slog.Logger,
) (string, context.CancelFunc, error) {
	tc, qc := GetQuicConfigFor(cert, alpn, logger)
	return RunQuicTestServer(tc, qc, doer, logger)
}

func GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func GetLoggerFor(app string) *slog.Logger {
	return GetLogger().With("app", app)
}

func NewTestCerts(testDir string, hosts []string) (map[string]string, error) {
	cfs := map[string]string{
		"cac": filepath.Join(testDir, "cacert.pem"),
		"cak": filepath.Join(testDir, "cacert-key.pem"),
		"svc": filepath.Join(testDir, "svcert.pem"),
		"svk": filepath.Join(testDir, "svcert-key.pem"),
		"clc": filepath.Join(testDir, "clcert.pem"),
		"clk": filepath.Join(testDir, "clcert-key.pem"),
	}
	if err := NewCaCertFiles(cfs["cac"], cfs["cak"]); err != nil {
		return nil, err
	}
	if err := NewCertFiles(hosts, cfs["cac"], cfs["cak"], cfs["svc"], cfs["svk"]); err != nil {
		return nil, err
	}
	if err := NewCertFiles(nil, cfs["cac"], cfs["cak"], cfs["clc"], cfs["clk"]); err != nil {
		return nil, err
	}
	return cfs, nil
}
