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

func (fc *fcClient) operate(fName string) error {
	var rsp FuncOperateRespMsg
	err := fc.ac.RunSyncFunction(fName, &FuncOperateReqMsg{fc.id}, &rsp)
	if err != nil {
		return err
	}
	if rsp.Error != "" {
		return errors.New(rsp.Error)
	}
	return nil

}
func (fc *fcClient) Run() error {
	return fc.operate(FnameFuncRun)
}

func (fc *fcClient) Start() error {
	return fc.operate(FnameFuncStart)
}

func (fc *fcClient) Wait() error {
	return fc.operate(FnameFuncWait)
}

func (fc *fcClient) Terminate() error {
	return fc.operate(FnameFuncTerminate)
}

type AppClient interface {
	AddIStream(id string) (OStream, error)
	GetOStream(id string) (IStream, error)
	RunSyncFunction(fName string, in any, out any) error
	NewFunction(fName string, id string) (FcClient, error)
	GetFunction(id string) FcClient
}

type appClient struct {
	cnc    Connection
	ctlMux sync.Mutex
	dMux   sync.Mutex
	funcs  map[string]FcClient
}

var _ AppClient = &appClient{}

func (ac *appClient) reqRoundTrip(cmd string, stId, funcId string, subProcess func(*appClient) error) error {
	ac.ctlMux.Lock()
	defer ac.ctlMux.Unlock()
	crqm := CtrlReqMsg{Command: cmd, StreamId: stId, FName: funcId}
	bs, err := json.Marshal(crqm)
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

	if subProcess != nil {
		if err := subProcess(ac); err != nil {
			return err
		}
	}

	bs = make([]byte, 4)
	if _, err = io.ReadFull(stream, bs); err != nil {
		return err
	}
	bln := binary.BigEndian.Uint32(bs)
	if bln > MaxRspSize {
		return fmt.Errorf("response too large (%d > %d)", bln, MaxRspSize)
	}
	bs = make([]byte, bln)
	_, err = io.ReadFull(stream, bs)
	crsm := CtrlRspMsg{}
	if err := json.Unmarshal(bs, &crsm); err != nil {
		return err
	}
	if crsm.Error != "" {
		return errors.New(crsm.Error)
	}
	return nil
}

func (ac *appClient) AddIStream(id string) (OStream, error) {
	var os OStream
	err := ac.reqRoundTrip(CmdAddOStream, id, "", func(client *appClient) error {
		var iErr error
		os, iErr = client.cnc.AddOStream(id)
		return iErr
	})
	return os, err
}

func (ac *appClient) GetOStream(id string) (IStream, error) {
	var is IStream
	err := ac.reqRoundTrip(CmdAddIStream, id, "", func(client *appClient) error {
		var iErr error
		is, iErr = client.cnc.AddIStream(id)
		return iErr
	})
	return is, err
}

func (ac *appClient) RunSyncFunction(fName string, in any, out any) error {
	err := ac.reqRoundTrip(
		CmdRunSyncFunction,
		"",
		fName,
		func(client *appClient) error {
			bs, err := json.Marshal(in)
			if err != nil {
				return err
			}
			wbs := make([]byte, len(bs)+4)
			binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
			copy(wbs[4:], bs)
			stream := ac.cnc.GetSyncStream()
			if _, err = stream.Write(wbs); err != nil {
				return err
			}

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
	ac.dMux.Lock()
	defer ac.dMux.Unlock()
	_, ok := ac.funcs[id]
	if ok {
		return nil, fmt.Errorf("function %s id %s already exists", fName, id)
	}
	var rsp NewFunctionRespMsg
	err := ac.RunSyncFunction(FNameNewFunction, &NewFunctionReqMsg{fName, id}, &rsp)
	if err != nil {
		return nil, err
	}
	if rsp.Error != "" {
		return nil, errors.New(rsp.Error)
	}
	fc := &fcClient{ac, &(rsp.Desc), id}
	ac.funcs[id] = fc
	return fc, nil
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
	if err := cnc.SetSyncStream(); err != nil {
		return nil, err
	}
	return &appClient{cnc: cnc, funcs: make(map[string]FcClient)}, nil
}
