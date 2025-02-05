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
	RunFunction(funcId string) error
	GetLogger() *slog.Logger
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
	crqm := CtrlReqMsg{}
	if err := json.Unmarshal(bs, &crqm); err != nil {
		return err
	}
	ac.GetLogger().Info("Received request", "req", crqm)
	switch crqm.Command {
	case CmdRunSyncFunction:
		return ac.RunFunction(crqm.FunctionId)
	default:
		return fmt.Errorf("unknown command: %s", crqm.Command)
	}
}

func (ac *appServerCnc) Close(err error) error {
	sErr := ""
	if err != nil {
		sErr = err.Error()
	}
	return ac.cnc.GetQuicConnection().CloseWithError(0, sErr)
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

func (ac *appServerCnc) sendRsp(err error) error {
	sErr := ""
	if err != nil {
		sErr = err.Error()
	}
	crsm := CtrlRspMsg{Error: sErr}
	bs, err := json.Marshal(crsm)
	if err != nil {
		return err
	}
	wbs := make([]byte, len(bs)+4)
	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
	copy(wbs[4:], bs)
	stream := ac.cnc.GetCtrlStream()
	if _, err = stream.Write(wbs); err != nil {
		return err
	}
	return nil
}

func (ac *appServerCnc) RunFunction(funcId string) error {
	var (
		err error
		fw  stf.Function
	)
	defer func() {
		if iErr := ac.sendRsp(err); iErr != nil {
			err = iErr
		}
	}()
	_ = fw
	fw, err = stf.NewSyncFuncWrapper(
		ac.cnc.GetCtx(),
		func(in any) any {
			return fmt.Sprintf("RunSyncFunction: %s(%s)", funcId, in)
		},
		ac.cnc.GetSyncStream(),
		ac.cnc.GetSyncStream(),
	)
	if err != nil {
		return err
	}
	if err = fw.Run(); err != nil {
		return err
	}
	return nil
}

func (ac *appServerCnc) GetLogger() *slog.Logger {
	return ac.cnc.GetLogger()
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
	if err := cnc.SetSyncStream(); err != nil {
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
