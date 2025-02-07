package netutils

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"log/slog"
	"os"
	"time"
)

func RunTestServer(alpn string,
	doer func(ctx context.Context, connection quic.Connection, logger *slog.Logger),
	logger *slog.Logger,
) (string, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cert, err := SelfSigned("localhost")
	if err != nil {
		cancel()
		return "", nil, err
	}
	var port string
	go func() {
		logger := logger
		listener, host, iport, ierr := GetQuicListener(":0", cert, alpn, logger)
		fmt.Fprintf(os.Stderr, "RunTestServer: listening on %s:%s\n", host, iport)
		if ierr != nil {
			err = ierr
			return
		}
		port = iport
		for {
			cnc, ierr := listener.Accept(ctx)
			if ierr != nil {
				fmt.Fprintf(os.Stderr, "RunTestServer: error accepting connection: %s\n", ierr)
				return
			}
			fmt.Fprintf(os.Stderr, "RunTestServer: new connection: %s\n", cnc.RemoteAddr().String())
			doer(ctx, cnc, logger)
		}
	}()
	time.Sleep(100 * time.Millisecond)
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
