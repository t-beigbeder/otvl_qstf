package appctl

import (
	"context"
	"crypto/tls"
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
	AddIStream(id string) error
	//GetOStream(id string) error
	RunSyncFunction(funcId string) error
	GetLogger() *slog.Logger
}

type appServerCnc struct {
	cnc Connection
	cat *FunctionCatalog
}

func (ac *appServerCnc) recvCtrl() (rid uint64, cmd string, req any, payload []byte, err error) {
	stream := ac.cnc.GetCtrlStream()
	bs := make([]byte, 12)
	if _, err = io.ReadFull(stream, bs); err != nil {
		return
	}
	rid = RidBs(bs[:]).Get()
	lCmd := LenBs(bs[8:]).Get()
	if lCmd > MaxReqSize {
		err = fmt.Errorf("request too large (%d > %d)", lCmd, MaxReqSize)
		return
	}
	bs = make([]byte, lCmd+4)
	if _, err = io.ReadFull(stream, bs); err != nil {
		return
	}
	cmd = string(bs[:lCmd])
	lnRq := LenBs(bs[lCmd:]).Get()
	if lnRq > MaxReqSize {
		err = fmt.Errorf("request too large (%d > %d)", lnRq, MaxReqSize)
		return
	}
	bs = make([]byte, lnRq)
	if _, err = io.ReadFull(stream, bs); err != nil {
		return
	}
	rd := GetReqDesc(cmd)
	if rd == nil {
		err = fmt.Errorf("unknown command: %s", cmd)
		return
	}
	req = rd.Req()
	if err = json.Unmarshal(bs, &req); err != nil {
		return
	}
	return
}

func (ac *appServerCnc) Handle() error {
	rid, cmd, areq, payload, err := ac.recvCtrl()
	if err != nil {
		return err
	}
	_, _, _, _ = rid, cmd, areq, payload
	ac.GetLogger().Info("Received request", "cmd", cmd, "req", areq)
	switch cmd {
	case CmdAddOStream:
		go ac.goSub(cmd, rid, ac.getOStream(rid, areq))
	default:
		ac.GetLogger().Error("Unknown command", "cmd", cmd)
		return fmt.Errorf("unknown command: %s", cmd)
	}
	return nil
}

func (ac *appServerCnc) Close(err error) error {
	sErr := ""
	if err != nil {
		sErr = err.Error()
	}
	return ac.cnc.GetQuicConnection().CloseWithError(0, sErr)
}

//func (ac *appServerCnc) sendRsp(err error) error {
//	sErr := ""
//	if err != nil {
//		sErr = err.Error()
//	}
//	crsm := CtrlRspMsg{Error: sErr}
//	bs, err := json.Marshal(crsm)
//	if err != nil {
//		return err
//	}
//	wbs := make([]byte, len(bs)+4)
//	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
//	copy(wbs[4:], bs)
//	stream := ac.cnc.GetCtrlStream()
//	if _, err = stream.Write(wbs); err != nil {
//		return err
//	}
//	return nil
//}
//
//func (ac *appServerCnc) runAndResp(toRun func(ac *appServerCnc) error) error {
//	err := toRun(ac)
//	eErr := ac.sendRsp(err)
//	if err == nil {
//		err = eErr
//	}
//	return err
//}

func (ac *appServerCnc) runAndResp(f func(asc *appServerCnc) error) error {
	return nil
}

func (ac *appServerCnc) AddIStream(id string) error {
	return ac.runAndResp(func(asc *appServerCnc) error {
		_, err := asc.cnc.AddIStream(id)
		return err
	})
}

func (ac *appServerCnc) goSub(cmd string, rid uint64, err error) {
	ac.GetLogger().Debug("Command done", "cmd", cmd, "rid", rid, "err", err)
	if err != nil {
		ac.GetLogger().Error("Error running command", "cmd", cmd, "rid", rid, "err", err)
	}
}

func (ac *appServerCnc) sendCtrl(rid uint64, rsp any, payload []byte) error {
	js, err := json.Marshal(rsp)
	if err != nil {
		return err
	}
	ln := 12 + len(js)
	if payload != nil {
		ln += 4 + len(payload)
	}
	bs := make([]byte, ln)
	SetRidBs(rid, bs)
	SetLenBs(uint32(len(js)), bs[8:])
	copy(bs[12:], js)
	if payload != nil {
		SetLenBs(uint32(len(payload)), bs[12+len(js):])
		copy(bs[16+len(js):], payload)
	}
	_, err = ac.cnc.GetCtrlStream().Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (ac *appServerCnc) getOStream(rid uint64, areq any) error {
	req, _ := areq.(*AddStreamReqMsg)
	_, err := ac.cnc.AddOStream(req.StreamId)
	rsp := &RespMsg{}
	if err != nil {
		rsp.Error = err.Error()
	}
	err = ac.sendCtrl(rid, rsp, nil)
	if err != nil {
		return err
	}
	return nil
}

func (ac *appServerCnc) RunSyncFunction(fName string) error {
	_, _, wf, _, err := ac.cat.GetFunction(fName)
	if err != nil {
		return err
	}
	if wf == nil {
		return fmt.Errorf("function %s has no wrapped function, currently not supported", fName)
	}
	var rr io.Reader
	var wr io.Writer
	return ac.runAndResp(func(asc *appServerCnc) error {
		fw, err := stf.NewSyncFuncWrapper(
			ac.cnc.GetCtx(),
			*wf,
			rr, wr,
			//ac.cnc.GetSyncStream(),
			//ac.cnc.GetSyncStream(),
		)
		if err != nil {
			return err
		}
		if err = fw.Run(); err != nil {
			return err
		}
		return nil
	})
}

func (ac *appServerCnc) GetLogger() *slog.Logger {
	return ac.cnc.GetLogger()
}

var _ AppServerCnc = &appServerCnc{}

type FcServer interface {
	GetFunction(id string) (stf.Function, error)
}

type AppServer interface {
	Catalog() *FunctionCatalog
	NewCnc(qc quic.Connection)
}

type appServer struct {
	ctx    context.Context
	cat    *FunctionCatalog
	logger *slog.Logger
	cncs   map[string]AppServerCnc
}

var _ AppServer = &appServer{}

func (as *appServer) Catalog() *FunctionCatalog {
	return as.cat
}

func (as *appServer) NewCnc(qc quic.Connection) {
	var err error
	id := qc.RemoteAddr().String()
	cnc := NewConnection(as.ctx, qc, id, true, true, as.logger)
	if err := cnc.SetCtrlStream(); err != nil {
		cnc.GetLogger().Error("AppServerConnectionHandler", "err", err)
		qc.CloseWithError(0, err.Error())
		return
	}
	//if err := cnc.SetSyncStream(); err != nil {
	//	cnc.GetLogger().Error("AppServerConnectionHandler", "err", err)
	//	qc.CloseWithError(0, err.Error())
	//	return
	//}
	ac := &appServerCnc{
		cnc: cnc,
		cat: as.cat,
	}
	as.cncs[id] = ac
	defer func() {
		ac.Close(err)
		delete(as.cncs, id)
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

func NewAppServer(ctx context.Context, cat *FunctionCatalog, logger *slog.Logger) AppServer {
	return &appServer{ctx: ctx, cat: cat, logger: logger, cncs: make(map[string]AppServerCnc)}
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
	as := NewAppServer(ctx, NewFunctionCatalog(), logger)
	for {
		qc, lErr := listener.Accept(ctx)
		if lErr != nil {
			logger.Info("RunAppServer: accept error", "err", lErr)
			continue
		}
		go as.NewCnc(qc)
	}
}
