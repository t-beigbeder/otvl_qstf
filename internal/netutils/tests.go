package netutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
		logger.Info("RunQuicTestServerWithCtc: listening", "host", ihost, "port", iport)
		if ierr != nil {
			err = ierr
			return
		}
		host, port = ihost, iport
		var checked bool
		for {
			cnc, ierr := listener.Accept(ctx)
			if ierr != nil {
				logger.Error("RunQuicTestServerWithCtc: accept error", "host", host, "port", iport, "err", ierr)
				return
			}
			if checked {
				logger.Info("RunQuicTestServerWithCtc: accepted connection", "host", host, "port", iport, "remoteAddr", cnc.RemoteAddr().String(), "checked", checked)
				doer(ctx, cnc, logger)
			} else {
				logger.Info("RunQuicTestServerWithCtc: accepted test check connection", "host", host, "port", iport, "remoteAddr", cnc.RemoteAddr().String(), "checked", checked)
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
		err = fmt.Errorf("RunQuicTestServerWithCtc: failed to connect to %s:%s err %v", host, port, lastErr)
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

func getCertDirs(testDir string, hosts []string) (string, string, string, string) {
	if os.Getenv("QSTF_TEST_CACHE") == "" {
		return testDir, testDir, testDir, testDir
	}
	tcd := os.Getenv("QSTF_TEST_CERTS_DIR")
	if tcd == "" {
		tcd = filepath.Join(os.TempDir(), "qstf_test_certs")
	}
	return filepath.Join(tcd, "ca"),
		filepath.Join(tcd, common.Hash(strings.Join(hosts, ","))),
		filepath.Join(tcd, "c1"),
		filepath.Join(tcd, "c2")
}

func makeCertIf(certDir string, maker func() error) error {
	if os.Getenv("QSTF_TEST_CACHE") == "" {
		return maker()
	}
	ffn := filepath.Join(certDir, "done.flag")
	if common.FileExists(ffn) {
		return nil
	}
	if !common.FileExists(certDir) {
		if err := os.MkdirAll(certDir, 0700); err != nil {
			return err
		}
	}
	if err := maker(); err != nil {
		return err
	}
	if err := common.WriteFile(ffn, nil); err != nil {
		return err
	}
	return nil
}

func NewTestCerts(testDir string, hosts []string, hasSecClient bool) (map[string]string, error) {
	caDir, svDir, c1Dir, c2Dir := getCertDirs(testDir, hosts)
	cfs := map[string]string{
		"cac": filepath.Join(caDir, "cacert.pem"),
		"cak": filepath.Join(caDir, "cacert-key.pem"),
		"svc": filepath.Join(svDir, "svcert.pem"),
		"svk": filepath.Join(svDir, "svcert-key.pem"),
		"c1c": filepath.Join(c1Dir, "c1cert.pem"),
		"c1k": filepath.Join(c1Dir, "c1cert-key.pem"),
	}
	if err := makeCertIf(caDir, func() error {
		return NewCaCertFiles(cfs["cac"], cfs["cak"])
	}); err != nil {
		return nil, err
	}
	if err := makeCertIf(svDir, func() error {
		return NewCertFiles(hosts, cfs["cac"], cfs["cak"], cfs["svc"], cfs["svk"])
	}); err != nil {
		return nil, err
	}
	if err := makeCertIf(c1Dir, func() error {
		return NewCertFiles(nil, cfs["cac"], cfs["cak"], cfs["c1c"], cfs["c1k"])
	}); err != nil {
		return nil, err
	}
	if !hasSecClient {
		return cfs, nil
	}
	cfs["c2c"] = filepath.Join(c2Dir, "c2cert.pem")
	cfs["c2k"] = filepath.Join(c2Dir, "c2cert-key.pem")
	if err := makeCertIf(c2Dir, func() error {
		return NewCertFiles(nil, cfs["cac"], cfs["cak"], cfs["c2c"], cfs["c2k"])
	}); err != nil {
		return nil, err
	}
	return cfs, nil
}
