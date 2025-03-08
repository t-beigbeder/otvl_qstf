package netutils

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/quic-go/quic-go"
	"time"
)

// NewQuicConn creates a Quic connection to host:port given by addr
func NewQuicConn(addr string, timeout time.Duration, tc *tls.Config, qc *quic.Config) (quic.Connection, error) {
	if tc == nil {
		return nil, errors.New("NewQuicConn: *tls.Config is nil")
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if qc == nil {
		qc = &quic.Config{
			KeepAlivePeriod: 20 * time.Second,
		}
	}
	conn, err := quic.DialAddr(ctx, addr, tc, qc)
	return conn, err
}
