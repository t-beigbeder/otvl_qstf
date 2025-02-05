package appctl

import (
	"context"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"log/slog"
	"os"
)

func RunTestServer() (string, context.CancelFunc, error) {
	var as AppServer
	port, cancel, err := netutils.RunTestServer(
		QstfAlpn,
		func(ctx context.Context, qc quic.Connection, logger *slog.Logger) {
			if as == nil {
				as = NewAppServer(ctx, logger)
			}
			as.NewCnc(qc)
		}, GetLoggerFor("server"))
	return port, cancel, err
}

func GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func GetLoggerFor(app string) *slog.Logger {
	return GetLogger().With("app", app)
}
