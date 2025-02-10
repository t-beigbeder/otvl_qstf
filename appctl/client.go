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
	GetDesc() *FunctionDesc
	GetId() string
	AddIStream(IStream) error
	AddOStream(OStream) error
	Run() error
	Start() error
	Wait() error
	Terminate() error
}

type fcClient struct {
	ac   *appClient
	desc *FunctionDesc
	id   string
}

func (fc *fcClient) GetDesc() *FunctionDesc {
	return fc.desc
}

func (fc *fcClient) GetId() string {
	return fc.id
}

func (fc *fcClient) AddIStream(is IStream) error {
	panic("implement me")
}

func (fc *fcClient) AddOStream(os OStream) error {
	panic("implement me")
}

func (fc *fcClient) Run() error {
	panic("implement me")
}

func (fc *fcClient) Start() error {
	panic("implement me")
}

func (fc *fcClient) Wait() error {
	panic("implement me")
}

func (fc *fcClient) Terminate() error {
	panic("implement me")
}

type AppClient interface {
	AddIStream(id string) (OStream, error)
	GetOStream(id string) (IStream, error)
	RunSyncFunction(fName string, in any, out any) error
	NewFunction(fName string, id string) (FcClient, error)
	GetFunction(id string) FcClient
}

type appClient struct {
	ctx        context.Context
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
		ac.logger.Error("Read ctrl stream header error: %v", "err", err)
		return
	}
	ridBs := RidBs(bs[:])
	rid = ridBs.Get()
	lnRs = LenBs(bs[8:]).Get()
	if lnRs > MaxRspSize {
		ac.logger.Error("Read rsp too large (%d)", "lnRs", lnRs)
		return
	}
	lnPl = LenBs(bs[12:]).Get()
	if lnPl > MaxRspSize {
		ac.logger.Error("Read rsp payload too large (%d)", "lnPl", lnPl)
		return
	}
	bs = make([]byte, lnRs)
	if _, err = io.ReadFull(stream, bs); err != nil {
		ac.logger.Error("Read ctrl stream rsp error: %v", "err", err)
		return
	}
	if lnPl != 0 {
		payload = make([]byte, lnPl)
		if _, err = io.ReadFull(stream, payload); err != nil {
			ac.logger.Error("Read ctrl stream rsp payload error: %v", "err", err)
			return
		}
	}
	ac.ctlMux.Lock()
	defer ac.ctlMux.Unlock()
	rc, ok := ac.rspChans[ridBs]
	if !ok {
		ac.logger.Info("rsp channel for rid %d not found", "rid", rid)
		return
	}
	if err = json.Unmarshal(bs, ac.rspRspVals[ridBs]); err != nil {
		ac.logger.Error("Read rsp json error: %v", "err", err)
		return
	}
	rc <- RspData{Rid: rid, Rsp: ac.rspRspVals[ridBs], Payload: payload}
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

func (ac *appClient) sendCtrl(rid uint64, cmd string, req any) error {
	js, err := json.Marshal(req)
	if err != nil {
		return err
	}
	bs := make([]byte, 8+4+len(cmd)+4+len(js))
	SetRidBs(rid, bs)
	SetLenBs(uint32(len(cmd)), bs[8:])
	copy(bs[12:], cmd)
	SetLenBs(uint32(len(js)), bs[12+len(cmd):])
	copy(bs[16+len(cmd):], js)
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
			err := ac.sendCtrl(rid, CmdAddOStream, req)
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

func (ac *appClient) oldRT(string, func(client *appClient) error) error {
	return nil
}

func (ac *appClient) oldAddIStream(id string) (OStream, error) {
	if id == "" {
		id = NextId("out")
	}
	var os OStream
	err := ac.oldRT(CmdAddOStream, func(client *appClient) error {
		var iErr error
		os, iErr = client.cnc.AddOStream(id)
		return iErr
	})
	return os, err
}

func (ac *appClient) GetOStream(id string) (IStream, error) {
	if id == "" {
		id = NextId("in")
	}
	var is IStream
	err := ac.oldRT(CmdAddIStream, func(client *appClient) error {
		var iErr error
		is, iErr = client.cnc.AddIStream(id)
		return iErr
	})
	return is, err
}

func (ac *appClient) RunSyncFunction(fName string, in any, out any) error {
	funcId := NextId(fmt.Sprintf("/%s/fc", fName))
	_ = funcId
	err := ac.oldRT(
		CmdRunSyncFunction,
		func(client *appClient) error {
			bs, err := json.Marshal(in)
			if err != nil {
				return err
			}
			wbs := make([]byte, len(bs)+4)
			binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
			copy(wbs[4:], bs)
			//stream := ac.cnc.GetSyncStream()
			//if _, err = stream.Write(wbs); err != nil {
			//	return err
			//}
			var stream io.Reader
			bs = make([]byte, 4)
			if _, err = io.ReadFull(stream, bs); err != nil {
				return err
			}
			bln := binary.BigEndian.Uint32(bs)
			if bln > MaxRspSize { // FIXME: depend on fName
				return fmt.Errorf("response too large (%d > %d)", bln, MaxRspSize)
			}
			bs = make([]byte, bln)
			_, err = io.ReadFull(stream, bs)
			if err := json.Unmarshal(bs, out); err != nil {
				return err
			}
			return nil
		})
	return err
}

func (ac *appClient) NewFunction(fName string, id string) (FcClient, error) {
	if id == "" {
		id = NextId(fmt.Sprintf("/%s/fc", fName))
	}
	//ac.dMux.Lock()
	//defer ac.dMux.Unlock()
	//_, ok := ac.funcs[id]
	//if ok {
	//	return nil, fmt.Errorf("function %s id %s already exists", fName, id)
	//}
	//var rsp NewFunctionRespMsg
	//err := ac.RunSyncFunction(FNameNewFunction, &NewFunctionReqMsg{fName, id}, &rsp)
	//if err != nil {
	//	return nil, err
	//}
	//if rsp.Error != "" {
	//	return nil, errors.New(rsp.Error)
	//}
	//fc := &fcClient{ac, &(rsp.Desc), id}
	//ac.funcs[id] = fc
	return nil, nil
}

func (ac *appClient) GetFunction(id string) FcClient {
	fc, _ := ac.funcs[id]
	return fc
}

func NewAppClient(ctx context.Context, sAddr string, logger *slog.Logger) (AppClient, error) {
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
	cnc := NewConnection(ctx, qc, "client", false, false, logger)
	if err := cnc.SetCtrlStream(); err != nil {
		return nil, err
	}
	ac := &appClient{
		ctx:        ctx,
		reqChans:   make(map[RidBs]chan RspData),
		rspChans:   make(map[RidBs]chan RspData),
		rspRspVals: make(map[RidBs]any),
		cnc:        cnc,
		funcs:      make(map[string]FcClient),
		logger:     logger,
	}
	go ac.lstnCtrl()
	return ac, nil
}
