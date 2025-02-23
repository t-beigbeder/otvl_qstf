package netutils

import (
	"context"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"time"
)

func GetQuicConn(addr string, alpn string, timeout time.Duration) (quic.Connection, error) {
	if timeout == 0 {
		timeout = time.Second * 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := quic.DialAddr(ctx, addr,
		GetUnsafeTlsConfigClient(alpn), // TODO: configure TLS
		&quic.Config{
			KeepAlivePeriod: 20 * time.Second,
			Tracer:          qlog.DefaultConnectionTracer,
		},
	)
	return conn, err
}
