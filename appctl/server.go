package appctl

import (
	"context"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"log/slog"
)

type AppServer interface {
	AddIStream(id string) (IStream, error)
	GetOStream(id string) (OStream, error)
	GetFunction(id string, iss []IStream, oss []OStream) (FcServer, error)
}

type appServer struct {
	cnc Connection
}

func (as *appServer) AddIStream(id string) (IStream, error) {
	//TODO implement me
	panic("implement me")
}

func (as *appServer) GetOStream(id string) (OStream, error) {
	//TODO implement me
	panic("implement me")
}

func (as *appServer) GetFunction(id string, iss []IStream, oss []OStream) (FcServer, error) {
	//TODO implement me
	panic("implement me")
}

var _ AppServer = &appServer{}

type FcServer interface {
	GetFunction(id string) (stf.Function, error)
}

func AppServerConnectionHandler(cnc Connection) {
	if err := cnc.SetCtrlStream(); err != nil {
		cnc.GetLogger().Error("AppServerConnectionHandler", "err", err)
	}
	for {
		select {
		case <- cnc.GetCtx().Done():
			return
			default:
				cnc.GetCtrlStream().
		}
	}
}

func quicAppServerConnectionHandler(ctx context.Context, qc quic.Connection, logger *slog.Logger) {
	cnc := &connection{
		ctx:          ctx,
		qc:           qc,
		id:           "server",
		isQuicServer: true,
		isAppServer:  true,
		logger:       logger,
	}
	logger.Info("AppServerConnectionHandler", "id", cnc.id)
	AppServerConnectionHandler(cnc)
}

func RunAppServer(
	ctx context.Context,
	addr string, cert *tls.Certificate,
	logger *slog.Logger,
) error {
	listener, host, port, err := netutils.GetQuicListener(addr, cert, QstfAlpn, logger)
	if err != nil {
		return err
	}
	logger.Info("RunAppServer: listening", "host", host, "port", port)
	for {
		qc, lErr := listener.Accept(ctx)
		if lErr != nil {
			logger.Info("RunAppServer: accept error", "err", lErr)
			continue
		}
		go quicAppServerConnectionHandler(ctx, qc, logger)
	}
}
