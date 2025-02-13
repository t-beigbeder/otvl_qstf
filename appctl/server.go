package appctl

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"io"
	"log/slog"
	"sync"
)

type AppServerCnc interface {
	Handle() error
	GetLogger() *slog.Logger
}

type appServerCnc struct {
	mux    sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	cnc    Connection
	cat    *FunctionCatalog
	err    error
	funcs  map[string]stf.Function
	fds    map[string]*FunctionDesc
	sthss  map[string]*StreamHandlers
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
	if lnPl > MaxInPlSize {
		err = fmt.Errorf("request too large (%d > %d)", lnPl, MaxInPlSize)
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
	case CmdCloseStream:
		ac.controlWorkload(cmd, rid, areq, nil, closeStream)
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

func (ac *appServerCnc) close() error {
	se := QServerStreamNoError
	if ac.err != nil {
		se = QServerStreamProtoError
	}
	for _, fc := range ac.funcs {
		fc.Close()
	}
	for _, is := range ac.cnc.GetIStreams() {
		is.QStream().CancelRead(se)
	}
	for _, os := range ac.cnc.GetOStreams() {
		os.QStream().CancelWrite(se)
	}
	sErr := ""
	aErr := QServerCloseNoError
	if ac.err != nil {
		aErr = QServerProtoError
		sErr = ac.err.Error()
	}
	return ac.cnc.GetQuicConnection().CloseWithError(aErr, sErr)
}

func (ac *appServerCnc) controlWorkload(cmd string, rid uint64, areq any, payload []byte, workload func(ac *appServerCnc, rid uint64, areq any, payload []byte) (any, []byte)) {
	go func() {
		arsp, payload := workload(ac, rid, areq, payload)
		err := ac.sendCtrl(rid, arsp, payload)
		if err != nil {
			ac.GetLogger().Error("Failed to send control response for workload", "cmd", cmd, "rid", rid, "err", err)
			ac.cnc.GetCtrlStream().QStream().CancelRead(QServerStreamProtoError)
			ac.err = errors.Join(ac.err, err)
		}
	}()
}

func (ac *appServerCnc) sendCtrl(rid uint64, rsp any, payload []byte) error {
	js, err := json.Marshal(rsp)
	if err != nil {
		return err
	}
	if len(js) > MaxRspSize {
		return fmt.Errorf("response too large (%d > %d)", len(js), MaxRspSize)
	}
	if len(payload) > MaxOutPlSize {
		return fmt.Errorf("response payload too large (%d > %d)", len(payload), MaxOutPlSize)
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

func closeStream(ac *appServerCnc, _ uint64, areq any, _ []byte) (any, []byte) {
	var err error
	req, _ := areq.(*CloseStreamReqMsg)
	rsp := &RespMsg{}
	if req.IsIn {
		for fcId, fc := range ac.funcs {
			for _, is := range fc.GetInStreams() {
				if is.GetName() == req.StreamId {
					err = errors.Join(err, fmt.Errorf("istream id %s owned by function %s", req.StreamId, fcId))
				}
			}
		}
	} else {
		for fcId, fc := range ac.funcs {
			for _, os := range fc.GetOutStreams() {
				if os.GetName() == req.StreamId {
					err = errors.Join(err, fmt.Errorf("ostream id %s owned by function %s", req.StreamId, fcId))
				}
			}
		}
	}
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
	rsp := &RunSyncFunctionRespMsg{}
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
	inBs := make([]byte, len(rqPl)+4)
	lnBs := NewLenBs(uint32(len(rqPl)))
	copy(inBs[0:], lnBs[:])
	copy(inBs[4:], rqPl)
	in := bytes.NewReader(inBs)
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
	err = ac.newFunction(req.FuncId, fc, fd, nil)
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	defer ac.delFunction(req.FuncId)
	err = fc.Run()
	if err != nil {
		rsp.Error = err.Error()
		return
	}
	outBs := out.Bytes()
	oln := LenBs(outBs[0:4]).Get()
	if int(oln)+4 != len(outBs) {
		rsp.Error = fmt.Sprintf("output payload len mismatch %d != %d", oln+4, len(outBs))
		return
	}
	rsp.Desc = *fd
	rsPl = outBs[4:]
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
	err = ac.newFunction(req.FuncId, fc, fd, sths)

	rsp.Desc = *fd
	return rsp, nil
}

func funcAddIStream(ac *appServerCnc, _ uint64, areq any, _ []byte) (arsp any, _ []byte) {
	req, _ := areq.(*FuncAddStreamReqMsg)
	rsp := &RespMsg{}
	arsp = rsp
	fc, fd, sths := ac.getFunction(req.FuncId)
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
	fc, fd, sths := ac.getFunction(req.FuncId)
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
	fc, fd, _ := ac.getFunction(req.FuncId)
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
	case "close":
		err = ac.delFunction(req.FuncId)
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
	cCtx, cancel := context.WithCancel(as.ctx)
	cnc := NewConnection(cCtx, qc, id, true, true, as.logger)
	if err := cnc.SetCtrlStream(); err != nil {
		cnc.GetLogger().Error("AppServerConnectionHandler", "err", err)
		qc.CloseWithError(QServerInitError, err.Error())
		return
	}
	ac := &appServerCnc{
		ctx:    cnc.GetCtx(),
		cancel: cancel,
		cnc:    cnc,
		cat:    as.cat,
		funcs:  make(map[string]stf.Function),
		fds:    make(map[string]*FunctionDesc),
		sthss:  make(map[string]*StreamHandlers),
	}
	as.cncs[id] = ac

	defer func() {
		ac.close()
		delete(as.cncs, id)
	}()

	for {
		select {
		case <-cnc.GetCtx().Done():
			return
		default:
			if err = ac.Handle(); err != nil {
				ac.err = errors.Join(ac.err, err)
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
