package appctl

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"io"
	"log/slog"
	"sync"
)

type FcClient interface {
	GetDesc() FunctionDesc
	GetId() string
	AddIStream(IStream) error
	AddOStream(OStream) error
	Run() error
	Start() error
	Wait() error
	Terminate() error
}

type fcClient struct {
	ac    *appClient
	desc  FunctionDesc
	id    string
	isIds []string
	osIds []string
}

func (fc *fcClient) GetDesc() FunctionDesc {
	return fc.desc
}

func (fc *fcClient) GetId() string {
	return fc.id
}

func (fc *fcClient) AddIStream(is IStream) error {
	fd := fc.desc
	if len(fc.isIds) >= len(fd.IStreams) {
		return fmt.Errorf("cannot add more than %d istreams", len(fd.IStreams))
	}
	isd := fd.IStreams[len(fc.isIds)]
	req := FuncAddStreamReqMsg{
		FuncId:   fc.id,
		StreamId: is.Id(),
		Discrete: isd.Discrete,
		MaxLen:   isd.MaxLen,
		MaxNb:    isd.MaxNb,
		BSize:    isd.BSize,
	}
	rsp := RespMsg{}
	ac := fc.ac
	rqDc := ac.launchBg(
		func(rid uint64, req, rqPl, rsp, rspPl any) error {
			err := ac.sendCtrl(rid, CmdFuncAddIStream, req, nil)
			if err != nil {
				return err
			}
			return nil
		},
		&req, nil, &rsp, nil,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return rspData.Err
	}
	arsp, ok := rspData.Rsp.(*RespMsg)
	if !ok {
		return fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return errors.New(arsp.Error)
	}
	return nil
}

func (fc *fcClient) AddOStream(os OStream) error {
	fd := fc.desc
	if len(fc.osIds) >= len(fd.OStreams) {
		return fmt.Errorf("cannot add more than %d ostreams", len(fd.OStreams))
	}
	osd := fd.OStreams[len(fc.osIds)]
	req := FuncAddStreamReqMsg{
		FuncId:   fc.id,
		StreamId: os.Id(),
		Discrete: osd.Discrete,
		MaxLen:   osd.MaxLen,
		MaxNb:    osd.MaxNb,
	}
	rsp := RespMsg{}
	ac := fc.ac
	rqDc := ac.launchBg(
		func(rid uint64, req, rqPl, rsp, rspPl any) error {
			err := ac.sendCtrl(rid, CmdFuncAddOStream, req, nil)
			if err != nil {
				return err
			}
			return nil
		},
		&req, nil, &rsp, nil,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return rspData.Err
	}
	arsp, ok := rspData.Rsp.(*RespMsg)
	if !ok {
		return fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return errors.New(arsp.Error)
	}
	return nil
}

func (fc *fcClient) funcOper(oper string) error {
	req := FuncOperReqMsg{Oper: oper, FuncId: fc.id}
	rsp := RespMsg{}
	ac := fc.ac
	rqDc := ac.launchBg(
		func(rid uint64, req, rqPl, rsp, rspPl any) error {
			err := ac.sendCtrl(rid, CmdFuncOper, req, nil)
			if err != nil {
				return err
			}
			return nil
		},
		&req, nil, &rsp, nil,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return rspData.Err
	}
	arsp, ok := rspData.Rsp.(*RespMsg)
	if !ok {
		return fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return errors.New(arsp.Error)
	}
	return nil

}

func (fc *fcClient) Run() error {
	return fc.funcOper("run")
}

func (fc *fcClient) Start() error {
	return fc.funcOper("start")
}

func (fc *fcClient) Wait() error {
	return fc.funcOper("wait")
}

func (fc *fcClient) Terminate() error {
	return fc.funcOper("terminate")
}

type AppClient interface {
	AddIStream(id string) (OStream, error)
	GetOStream(id string) (IStream, error)
	GetFDesc(fName string) (*FunctionDesc, error)
	RunSyncFunction(fName string, id string, in any, out any) error
	NewFunction(fName string, id string) (FcClient, error)
	GetFunction(id string) FcClient
}

type appClient struct {
	ctx        context.Context
	cancel     context.CancelFunc
	ctlMux     sync.Mutex
	curReqId   uint64
	reqChans   map[RidBs]chan RspData
	rspChans   map[RidBs]chan RspData
	rspRspVals map[RidBs]any
	cnc        Connection
	funcs      map[string]FcClient
	logger     *slog.Logger
}

var _ AppClient = &appClient{}

func (ac *appClient) recvCtrl() {
	var (
		err     error
		rid     uint64
		lnRs    uint32
		lnPl    uint32
		payload []byte
	)
	stream := ac.cnc.GetCtrlStream()
	bs := make([]byte, 16)
	if _, err = io.ReadFull(stream, bs); err != nil {
		ac.logger.Error("Read ctrl stream header error", "err", err)
		ac.cancel()
		return
	}
	ridBs := RidBs(bs[:])
	rid = ridBs.Get()

	defer func() {
		ac.ctlMux.Lock()
		defer ac.ctlMux.Unlock()
		rc, ok := ac.rspChans[ridBs]
		if !ok {
			ac.logger.Info("rsp channel for rid not found", "rid", rid)
			return
		}
		rc <- RspData{Rid: rid, Rsp: ac.rspRspVals[ridBs], Payload: payload}

	}()

	lnRs = LenBs(bs[8:]).Get()
	if lnRs > MaxRspSize {
		ac.logger.Error("Read rsp too large", "lnRs", lnRs)
		return
	}
	lnPl = LenBs(bs[12:]).Get()
	if lnPl > MaxRspSize {
		ac.logger.Error("Read rsp payload too large", "lnPl", lnPl)
		return
	}
	bs = make([]byte, lnRs)
	if _, err = io.ReadFull(stream, bs); err != nil {
		ac.logger.Error("Read ctrl stream rsp error", "err", err)
		return
	}
	if lnPl != 0 {
		payload = make([]byte, lnPl)
		if _, err = io.ReadFull(stream, payload); err != nil {
			ac.logger.Error("Read ctrl stream rsp payload error", "err", err)
			return
		}
	}

	val, ok := ac.rspRspVals[ridBs]
	if !ok {
		ac.logger.Info("rsp values for rid not found", "rid", rid)
		return
	}
	if err = json.Unmarshal(bs, val); err != nil {
		ac.logger.Error("Read rsp json error", "err", err)
		return
	}
}

func (ac *appClient) lstnCtrl() {
	ac.logger.Info("starting ctrl listener")

	for {
		select {
		case <-ac.ctx.Done():
			ac.logger.Info("shutting down ctrl listener")
			return
		default:
			ac.recvCtrl()
		}
	}
}

func (ac *appClient) sendCtrl(rid uint64, cmd string, req any, payload []byte) error {
	js, err := json.Marshal(req)
	if err != nil {
		return err
	}
	bs := make([]byte, 8+4+4+4+len(cmd)+len(js)+len(payload))
	SetRidBs(rid, bs)
	SetLenBs(uint32(len(cmd)), bs[8:])
	SetLenBs(uint32(len(js)), bs[12:])
	SetLenBs(uint32(len(payload)), bs[16:])
	copy(bs[20:], cmd)
	copy(bs[20+len(cmd):], js)
	if payload != nil {
		copy(bs[20+len(cmd)+len(js):], payload)
	}
	_, err = ac.cnc.GetCtrlStream().Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (ac *appClient) waitResp(rid uint64) (*RspData, error) {
	ac.ctlMux.Lock()
	ridBs := NewRidBs(rid)
	rc, ok := ac.rspChans[ridBs]
	ac.ctlMux.Unlock()
	if !ok {
		return nil, fmt.Errorf("rid %d response channel not found", rid)
	}
	select {
	case <-ac.ctx.Done():
		return nil, errors.New("connection closed while waiting for response")
	default:
		rd := <-rc
		return &rd, nil
	}
}

func (ac *appClient) launchBg(
	workLoad func(rid uint64, req, rqPl, rsp, rspPl any) error,
	req, rqPl, rsp, rspPl any,
) chan RspData {
	ac.ctlMux.Lock()
	defer ac.ctlMux.Unlock()
	ac.curReqId++
	reqId := ac.curReqId
	var bs RidBs
	binary.BigEndian.PutUint64(bs[:], reqId)
	ac.reqChans[bs] = make(chan RspData, 1)
	ac.rspChans[bs] = make(chan RspData, 1)
	ac.rspRspVals[bs] = rsp

	go func() {
		var (
			err error
			rd  *RspData
		)
		defer func() {
			ac.reqChans[bs] <- RspData{Rid: reqId, Err: err, Rsp: rsp, Payload: rspPl}
		}()
		err = workLoad(reqId, req, rqPl, rsp, rspPl)
		if err != nil {
			return
		}
		rd, err = ac.waitResp(reqId)
		if err != nil {
			return
		}
		rsp = rd.Rsp
		rspPl = rd.Payload
	}()
	return ac.reqChans[bs]
}

func (ac *appClient) AddIStream(id string) (OStream, error) {
	if id == "" {
		id = NextId("out")
	}
	req := AddStreamReqMsg{StreamId: id}
	rsp := RespMsg{}
	var os OStream
	rqDc := ac.launchBg(
		func(rid uint64, req, _, rsp, _ any) error {
			err := ac.sendCtrl(rid, CmdAddOStream, req, nil)
			if err != nil {
				return err
			}
			os, err = ac.cnc.AddOStream(id)
			if err != nil {
				return err
			}
			return nil
		},
		&req, nil, &rsp, nil,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return nil, rspData.Err
	}
	arsp, ok := rspData.Rsp.(*RespMsg)
	if !ok {
		return nil, fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return nil, errors.New(arsp.Error)
	}
	return os, nil
}

func (ac *appClient) GetOStream(id string) (IStream, error) {
	if id == "" {
		id = NextId("in")
	}
	req := AddStreamReqMsg{StreamId: id}
	rsp := RespMsg{}
	var is IStream
	rqDc := ac.launchBg(
		func(rid uint64, req, _, rsp, _ any) error {
			err := ac.sendCtrl(rid, CmdAddIStream, req, nil)
			if err != nil {
				return err
			}
			is, err = ac.cnc.AddIStream(id)
			if err != nil {
				return err
			}
			return nil
		},
		&req, nil, &rsp, nil,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return nil, rspData.Err
	}
	arsp, ok := rspData.Rsp.(*RespMsg)
	if !ok {
		return nil, fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return nil, errors.New(arsp.Error)
	}
	return is, nil
}

func (ac *appClient) GetFDesc(fName string) (*FunctionDesc, error) {
	req := GetFDescReqMsg{FdName: fName}
	rsp := GetFDescRespMsg{}
	rqDc := ac.launchBg(
		func(rid uint64, req, _, rsp, _ any) error {
			err := ac.sendCtrl(rid, CmdGetFDesc, req, nil)
			if err != nil {
				return err
			}
			return nil
		},
		&req, nil, &rsp, nil,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return nil, rspData.Err
	}
	arsp, ok := rspData.Rsp.(*GetFDescRespMsg)
	if !ok {
		return nil, fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return nil, errors.New(arsp.Error)
	}
	return &arsp.Desc, nil
}

func (ac *appClient) RunSyncFunction(fName string, id string, in any, out any) error {
	if id == "" {
		id = NextId(fmt.Sprintf("/%s/fc", fName))
	}
	req := RunSyncFunctionReqMsg{FdName: fName, FuncId: id}
	rsp := RunSyncFunctionRespMsg{}
	rqDc := ac.launchBg(
		func(rid uint64, req, rqPl, rsp, rspPl any) error {
			rqPlBs, err := json.Marshal(rqPl)
			if err != nil {
				return err
			}
			err = ac.sendCtrl(rid, CmdRunSyncFunction, req, rqPlBs)
			if err != nil {
				return err
			}
			return nil
		},
		&req, in, &rsp, out,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return rspData.Err
	}
	arsp, ok := rspData.Rsp.(*RunSyncFunctionRespMsg)
	if !ok {
		return fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return errors.New(arsp.Error)
	}

	if arsp.Desc.Wrapper.OutMarshaller == MarshalJson {
		plbs, ok := rspData.Payload.([]byte)
		if !ok {
			return fmt.Errorf("unexpected payload type: %T", rspData.Payload)
		}
		err := json.Unmarshal(plbs, out)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("payload output unmarshaller %d not yet supported", arsp.Desc.Wrapper.OutMarshaller)
	}
	return nil
}

func (ac *appClient) NewFunction(fName string, id string) (FcClient, error) {
	if id == "" {
		id = NextId(fmt.Sprintf("/%s/fc", fName))
	}
	req := NewFunctionReqMsg{FdName: fName, FuncId: id}
	rsp := NewFunctionRespMsg{}
	rqDc := ac.launchBg(
		func(rid uint64, req, rqPl, rsp, rspPl any) error {
			err := ac.sendCtrl(rid, CmdNewFunction, req, nil)
			if err != nil {
				return err
			}
			return nil
		},
		&req, nil, &rsp, nil,
	)
	rspData := <-rqDc
	if rspData.Err != nil {
		return nil, rspData.Err
	}
	arsp, ok := rspData.Rsp.(*NewFunctionRespMsg)
	if !ok {
		return nil, fmt.Errorf("unexpected rsp type: %T", rspData.Rsp)
	}
	if arsp.Error != "" {
		return nil, errors.New(arsp.Error)
	}
	fc := &fcClient{ac, rsp.Desc, id, nil, nil}
	ac.funcs[id] = fc
	return fc, nil
}

func (ac *appClient) GetFunction(id string) FcClient {
	fc, _ := ac.funcs[id]
	return fc
}

func (ac *appClient) close(err error) error {
	ac.logger.Info("close app client requested", "cn", ac.cnc, "err", err)
	for _, rqc := range ac.reqChans {
		close(rqc)
	}
	return err
}

func NewAppClient(pCtx context.Context, sAddr string, logger *slog.Logger) (AppClient, error) {
	var (
		qc  quic.Connection
		err error
	)
	qc, err = netutils.GetQuicConn(sAddr, QstfAlpn)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			qc.CloseWithError(0, "")
		}
	}()
	ctx, cancel := context.WithCancel(pCtx)
	cnc := NewConnection(ctx, qc, "client", false, false, logger)
	if err := cnc.SetCtrlStream(); err != nil {
		return nil, err
	}
	ac := &appClient{
		ctx:        ctx,
		cancel:     cancel,
		reqChans:   make(map[RidBs]chan RspData),
		rspChans:   make(map[RidBs]chan RspData),
		rspRspVals: make(map[RidBs]any),
		cnc:        cnc,
		funcs:      make(map[string]FcClient),
		logger:     logger,
	}
	go ac.lstnCtrl()
	go func() {
		for {
			select {
			case <-ac.ctx.Done():
				ac.close(errors.New("app client closed"))
				return
			}
		}
	}()
	return ac, nil
}
