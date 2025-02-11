package appctl

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"io"
	"log/slog"
)

type AppServerCnc interface {
	Handle() error
	Close(error) error
	GetLogger() *slog.Logger
}

type appServerCnc struct {
	ctx context.Context
	cnc Connection
	cat *FunctionCatalog
}

func (ac *appServerCnc) recvCtrl() (rid uint64, cmd string, req any, payload []byte, err error) {
	stream := ac.cnc.GetCtrlStream()
	hbs := make([]byte, 20)
	if _, err = io.ReadFull(stream, hbs); err != nil {
		return
	}
	rid = RidBs(hbs[:]).Get()
	lCmd := LenBs(hbs[8:]).Get()
	if lCmd > MaxReqSize {
		err = fmt.Errorf("request too large (%d > %d)", lCmd, MaxReqSize)
		return
	}
	lnRq := LenBs(hbs[12:]).Get()
	if lnRq > MaxReqSize {
		err = fmt.Errorf("request too large (%d > %d)", lnRq, MaxReqSize)
		return
	}
	lnPl := LenBs(hbs[16:]).Get()
	if lnPl > MaxReqSize {
		err = fmt.Errorf("request too large (%d > %d)", lnPl, MaxReqSize)
		return
	}

	data := make([]byte, lCmd+lnRq+lnPl)
	if _, err = io.ReadFull(stream, data); err != nil {
		return
	}
	cmd = string(data[:lCmd])
	rbq := data[lCmd : lCmd+lnRq]
	rd := GetReqDesc(cmd)
	if rd == nil {
		err = fmt.Errorf("unknown command: %s", cmd)
		return
	}
	req = rd.Req()
	if err = json.Unmarshal(rbq, &req); err != nil {
		return
	}
	if lnPl != 0 {
		payload = data[lCmd+lnRq:]
	}
	return
}

func (ac *appServerCnc) Handle() error {
	rid, cmd, areq, payload, err := ac.recvCtrl()
	if err != nil {
		return err
	}
	ac.GetLogger().Info("Received request", "cmd", cmd, "req", areq, "rid", rid)
	switch cmd {
	case CmdAddOStream:
		ac.controlWorkload(cmd, rid, areq, nil, getOStream)
	case CmdAddIStream:
		ac.controlWorkload(cmd, rid, areq, nil, addIStream)
	case CmdGetFDesc:
		ac.controlWorkload(cmd, rid, areq, nil, getFDesc)
	case CmdRunSyncFunction:
		ac.controlWorkload(cmd, rid, areq, payload, runSyncFunction)
	case CmdNewFunction:
		ac.controlWorkload(cmd, rid, areq, nil, newFunction)
	case CmdFuncAddIStream:
		ac.controlWorkload(cmd, rid, areq, nil, funcAddIStream)
	case CmdFuncAddOStream:
		ac.controlWorkload(cmd, rid, areq, nil, funcAddOStream)
	case CmdFuncOper:
		ac.controlWorkload(cmd, rid, areq, nil, funcOper)
	default:
		ac.GetLogger().Error("Unknown command", "cmd", cmd)
		return ac.respError(rid, fmt.Errorf("unknown command: %s", cmd))
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

func (ac *appServerCnc) controlWorkload(cmd string, rid uint64, areq any, payload []byte, workload func(ac *appServerCnc, rid uint64, areq any, payload []byte) (any, []byte)) {
	go func() {
		arsp, payload := workload(ac, rid, areq, payload)
		err := ac.sendCtrl(rid, arsp, payload)
		if err != nil {
			ac.GetLogger().Error("Failed to send control response for workload", "cmd", cmd, "rid", rid, "err", err)
		}
	}()
}

func (ac *appServerCnc) sendCtrl(rid uint64, rsp any, payload []byte) error {
	js, err := json.Marshal(rsp)
	if err != nil {
		return err
	}
	ln := 16 + len(js)
	if payload != nil {
		ln += len(payload)
	}
	bs := make([]byte, ln)
	SetRidBs(rid, bs)
	SetLenBs(uint32(len(js)), bs[8:])
	lpl := 0
	if payload != nil {
		lpl = len(payload)
	}
	SetLenBs(uint32(lpl), bs[12:])
	copy(bs[16:], js)
	if payload != nil {
		copy(bs[16+len(js):], payload)
	}
	_, err = ac.cnc.GetCtrlStream().Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (ac *appServerCnc) respError(rid uint64, err error) error {
	rsp := &RespMsg{Error: err.Error()}
	err = ac.sendCtrl(rid, rsp, nil)
	if err != nil {
		return err
	}
	return nil
}

func getOStream(ac *appServerCnc, _ uint64, areq any, _ []byte) (any, []byte) {
	req, _ := areq.(*AddStreamReqMsg)
	_, err := ac.cnc.AddOStream(req.StreamId)
	rsp := &RespMsg{}
	if err != nil {
		rsp.Error = err.Error()
	}
	return rsp, nil
}

func addIStream(ac *appServerCnc, _ uint64, areq any, _ []byte) (any, []byte) {
	req, _ := areq.(*AddStreamReqMsg)
	_, err := ac.cnc.AddIStream(req.StreamId)
	rsp := &RespMsg{}
	if err != nil {
		rsp.Error = err.Error()
	}
	return rsp, nil
}

func getFDesc(ac *appServerCnc, _ uint64, areq any, _ []byte) (any, []byte) {
	req, _ := areq.(*GetFDescReqMsg)
	fd, _, _, _, err := ac.cat.GetFunction(req.FdName)
	rsp := &GetFDescRespMsg{}
	if err != nil {
		rsp.Error = err.Error()
	}
	rsp.Desc = *fd
	return rsp, nil
}

func runSyncFunction(ac *appServerCnc, _ uint64, areq any, rqPl []byte) (arsp any, rsPl []byte) {
	req, _ := areq.(*RunSyncFunctionReqMsg)
	rsp := &RespMsg{}
	arsp = rsp
	fd, _, wf, _, err := ac.cat.GetFunction(req.FdName)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	if wf == nil {
		rsp.Error = fmt.Sprintf("function %s has no wrapped function, currently not supported", req.FdName)
		return
	}
	values := make(map[string]any)
	fcCtx := context.WithValue(ac.ctx, "values", values)
	in := bytes.NewReader(rqPl)
	out := bfio.NewBufWr()
	fc, err := stf.NewSyncFuncWrapper(
		fcCtx,
		*wf,
		in,
		out,
	)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	values["fc"] = fc
	err = ac.cnc.NewFunction(req.FuncId, fc, fd, nil)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	err = fc.Run()
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	rsPl, err = wf.Marshaller(out.Bytes())
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	return
}

func newFunction(ac *appServerCnc, _ uint64, areq any, _ []byte) (arsp any, _ []byte) {
	req, _ := areq.(*NewFunctionReqMsg)
	rsp := &NewFunctionRespMsg{}
	arsp = rsp
	fd, sw, wf, sths, err := ac.cat.GetFunction(req.FdName)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	if wf != nil {
		rsp.Error = fmt.Sprintf("Function %s is wrapped and must be run with RunSyncFunction", fd.Name)
		return
	}
	if sw == nil {
		rsp.Error = fmt.Sprintf("Function %s cannot be run as it doesn't have StartWaiter interface defined", fd.Name)
		return
	}
	opts := []stf.FcOption{stf.FcId(req.FuncId)}
	if fd.Terminable {
		opts = append(opts, stf.FcTerminable(true))
	}
	values := make(map[string]any)
	fcCtx := context.WithValue(ac.ctx, "values", values)
	fc, err := stf.NewFunction(fcCtx, sw, opts...)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	values["fc"] = fc
	err = ac.cnc.NewFunction(req.FuncId, fc, fd, sths)

	rsp.Desc = *fd
	return rsp, nil
}

func funcAddIStream(ac *appServerCnc, _ uint64, areq any, _ []byte) (arsp any, _ []byte) {
	req, _ := areq.(*FuncAddStreamReqMsg)
	rsp := &RespMsg{}
	arsp = rsp
	fc, fd, sths := ac.cnc.GetFunction(req.FuncId)
	if fc == nil || fd == nil {
		rsp.Error = fmt.Sprintf("Function %s does not exist", req.FuncId)
		return
	}
	is := ac.cnc.GetIStream(req.StreamId)
	if is == nil {
		rsp.Error = fmt.Sprintf("Stream in %s does not exist", req.StreamId)
		return
	}
	if len(fc.GetInStreams()) >= len(fd.IStreams) {
		rsp.Error = fmt.Sprintf("Stream in %s has no descriptor", req.StreamId)
		return
	}
	opts := []stf.IstOption{stf.IstId(req.StreamId)}
	stx := len(fc.GetInStreams())
	opts = append(opts, stf.IstDiscrete(fd.IStreams[stx].Discrete))
	opts = append(opts, stf.IstMaxNb(fd.IStreams[stx].MaxNb))
	isths := sths.ihs[stx]
	if isths.BSet != nil {
		opts = append(opts, stf.IstBSet(isths.BSet))
	}
	if isths.ASet != nil {
		opts = append(opts, stf.IstASet(isths.Unmarshal, isths.NewASet, isths.ASet))
	}
	_, err := fc.AddInStream(is, opts...)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	return rsp, nil
}

func funcAddOStream(ac *appServerCnc, _ uint64, areq any, _ []byte) (arsp any, _ []byte) {
	req, _ := areq.(*FuncAddStreamReqMsg)
	rsp := &RespMsg{}
	arsp = rsp
	fc, fd, sths := ac.cnc.GetFunction(req.FuncId)
	if fc == nil || fd == nil {
		rsp.Error = fmt.Sprintf("Function %s does not exist", req.FuncId)
		return
	}
	os := ac.cnc.GetOStream(req.StreamId)
	if os == nil {
		rsp.Error = fmt.Sprintf("Stream out %s does not exist", req.StreamId)
		return
	}
	if len(fc.GetOutStreams()) >= len(fd.OStreams) {
		rsp.Error = fmt.Sprintf("Stream out %s has no descriptor", req.StreamId)
		return
	}
	opts := []stf.OstOption{stf.OstId(req.StreamId)}
	stx := len(fc.GetOutStreams())
	opts = append(opts, stf.OstDiscrete(fd.OStreams[stx].Discrete))
	opts = append(opts, stf.OstMaxNb(fd.OStreams[stx].MaxNb))
	osths := sths.ohs[stx]
	if osths.BGet != nil {
		opts = append(opts, stf.OstBGet(osths.BGet))
	}
	if osths.AGet != nil {
		opts = append(opts, stf.OstAGet(osths.AGet, osths.Marshaller))
	}
	_, err := fc.AddOutStream(os, opts...)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	return rsp, nil
}

func funcOper(ac *appServerCnc, _ uint64, areq any, _ []byte) (arsp any, _ []byte) {
	req, _ := areq.(*FuncOperReqMsg)
	rsp := &RespMsg{}
	arsp = rsp
	fc, fd, _ := ac.cnc.GetFunction(req.FuncId)
	if fc == nil || fd == nil {
		rsp.Error = fmt.Sprintf("Function %s does not exist", req.FuncId)
		return
	}
	if len(fc.GetInStreams()) != len(fd.IStreams) {
		rsp.Error = fmt.Sprintf("instreams should be %d, got %d", len(fd.IStreams), len(fc.GetInStreams()))
		return
	}
	if len(fc.GetOutStreams()) != len(fd.OStreams) {
		rsp.Error = fmt.Sprintf("outstreams should be %d, got %d", len(fd.IStreams), len(fc.GetInStreams()))
		return
	}
	var err error
	switch req.Oper {
	case "run":
		err = fc.Run()
	case "start":
		err = fc.Start()
	case "wait":
		err = fc.Wait()
	case "terminate":
		fc.Terminate()
	default:
		err = fmt.Errorf("Oper %s not supported", req.Oper)
	}
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	return rsp, nil
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
	ac := &appServerCnc{
		ctx: cnc.GetCtx(),
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
