package appctl

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"io"
	"log/slog"
)

type AppServerCnc interface {
	Handle() error
	Close(error) error
	AddIStream(id string) (IStream, error)
	GetOStream(id string) (OStream, error)
	GetFunction(id string, iss []IStream, oss []OStream) (FcServer, error)
}

type appServerCnc struct {
	cnc Connection
}

func (ac *appServerCnc) Handle() error {
	stream := ac.cnc.GetCtrlStream()
	bs := make([]byte, 4)
	if _, err := io.ReadFull(stream, bs); err != nil {
		return err
	}
	bln := binary.BigEndian.Uint32(bs)
	if bln > MaxRspSize {
		return fmt.Errorf("response too large (%d > %d)", bln, MaxRspSize)
	}
	bs = make([]byte, bln)
	if _, err := io.ReadFull(stream, bs); err != nil {
		return err
	}
	ac.cnc.AddCtrlRead(int(bln + 4))
	crqm := CtrlReqMsg{}
	if err := json.Unmarshal(bs, &crqm); err != nil {
		return err
	}
	switch crqm.Command {
	case CmdRunFunction:
		return nil
	default:
		return fmt.Errorf("unknown command: %s", crqm.Command)
	}
}

func (ac *appServerCnc) Close(err error) error {
	return ac.cnc.GetQuicConnection().CloseWithError(0, err.Error())
}

func (ac *appServerCnc) AddIStream(id string) (IStream, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *appServerCnc) GetOStream(id string) (OStream, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *appServerCnc) GetFunction(id string, iss []IStream, oss []OStream) (FcServer, error) {
	//TODO implement me
	panic("implement me")
}

var _ AppServerCnc = &appServerCnc{}

type FcServer interface {
	GetFunction(id string) (stf.Function, error)
}

type AppServer interface {
	NewCnc(qc quic.Connection)
}

type appServer struct {
	ctx    context.Context
	logger *slog.Logger
	cncs   map[string]AppServerCnc
}

var _ AppServer = &appServer{}

func (as *appServer) NewCnc(qc quic.Connection) {
	var err error
	cnc := &connection{
		ctx:          as.ctx,
		qc:           qc,
		id:           qc.RemoteAddr().String(),
		isQuicServer: true,
		isAppServer:  true,
		logger:       as.logger,
	}
	if err := cnc.SetCtrlStream(); err != nil {
		cnc.GetLogger().Error("AppServerConnectionHandler", "err", err)
		cnc.qc.CloseWithError(0, err.Error())
		return
	}
	ac := &appServerCnc{
		cnc: cnc,
	}
	as.cncs[cnc.id] = ac
	defer func() {
		ac.Close(err)
		delete(as.cncs, cnc.id)
	}()
	for {
		select {
		case <-cnc.GetCtx().Done():
			return
		default:
			if err = ac.Handle(); err != nil {
				return
			}
		}
	}
}

func NewAppServer(ctx context.Context, logger *slog.Logger) AppServer {
	return &appServer{ctx: ctx, logger: logger, cncs: make(map[string]AppServerCnc)}
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
	as := NewAppServer(ctx, logger)
	for {
		qc, lErr := listener.Accept(ctx)
		if lErr != nil {
			logger.Info("RunAppServer: accept error", "err", lErr)
			continue
		}
		go as.NewCnc(qc)
	}
}
