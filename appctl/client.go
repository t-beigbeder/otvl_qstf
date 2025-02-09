package appctl

import (
	"context"
	"encoding/binary"
	"encoding/json"
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
	ctlMux   sync.Mutex
	curReqId uint64
	cnc      Connection
	funcs    map[string]FcClient
}

var _ AppClient = &appClient{}

func (ac *appClient) oldRT(string, func(client *appClient) error) error {
	return nil
}
func (ac *appClient) AddIStream(id string) (OStream, error) {
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
	return &appClient{cnc: cnc, funcs: make(map[string]FcClient)}, nil
}
