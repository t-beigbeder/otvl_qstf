package netutils

import (
	"context"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"time"
)

// NewQuicConn creates a Quic connection to host:port given by addr
func NewQuicConn(addr string, timeout time.Duration, tc *tls.Config, alpns []string, qc *quic.Config) (quic.Connection, error) {
	if timeout == 0 {
		timeout = time.Second * 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if tc == nil {
		tc = GetUnsafeTlsConfigClient(alpns)
	}
	if qc == nil {
		qc = &quic.Config{
			KeepAlivePeriod: 20 * time.Second,
		}
	}
	conn, err := quic.DialAddr(ctx, addr, tc, qc)
	return conn, err
}
