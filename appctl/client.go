package appctl

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"io"
	"sync"
)

type AppClient interface {
	AddIStream(id string) (OStream, error)
	GetOStream(id string) (IStream, error)
	RunFunction(id string, iss []OStream, oss []IStream) (FcClient, error)
	GetFunction(id string, iss []OStream, oss []IStream) (FcClient, error)
}

type FcClient interface {
	Run() error
	Start() error
	Wait() error
	Terminate()
	State() stf.FunctionState
	Options() stf.FcOptions
	Error() error
}

type fcClient struct {
	id string
}

func (fc *fcClient) Run() error {
	//TODO implement me
	panic("implement me")
}

func (fc *fcClient) Start() error {
	//TODO implement me
	panic("implement me")
}

func (fc *fcClient) Wait() error {
	//TODO implement me
	panic("implement me")
}

func (fc *fcClient) Terminate() {
	//TODO implement me
	panic("implement me")
}

func (fc *fcClient) State() stf.FunctionState {
	//TODO implement me
	panic("implement me")
}

func (fc *fcClient) Options() stf.FcOptions {
	//TODO implement me
	panic("implement me")
}

func (fc *fcClient) Error() error {
	//TODO implement me
	panic("implement me")
}

var _ FcClient = &fcClient{}

type appClient struct {
	cnc    Connection
	ctlMux sync.Mutex
}

var _ AppClient = &appClient{}

func (ac *appClient) reqRoundTrip(cmd string, funcId string) error {
	ac.ctlMux.Lock()
	defer ac.ctlMux.Unlock()
	crqm := CtrlReqMsg{Command: cmd, FunctionId: funcId}
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
	ac.cnc.AddCtrlWritten(len(wbs))

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
	ac.cnc.AddCtrlRead(int(bln + 4))
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
	//TODO implement me
	panic("implement me")
}

func (ac *appClient) GetOStream(id string) (IStream, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *appClient) RunFunction(funcId string, iss []OStream, oss []IStream) (FcClient, error) {
	err := ac.reqRoundTrip(CmdRunFunction, funcId)
	if err != nil {
		return nil, err
	}
	return &fcClient{id: funcId}, nil
}

func (ac *appClient) GetFunction(id string, iss []OStream, oss []IStream) (FcClient, error) {
	//TODO implement me
	panic("implement me")
}

func NewAppClient(ctx context.Context, sAddr string) (AppClient, error) {
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
	cnc := connection{
		ctx: ctx,
		qc:  qc,
		id:  "client",
	}
	if err := cnc.SetCtrlStream(); err != nil {
		return nil, err
	}
	if err := cnc.SetSyncStream(); err != nil {
		return nil, err
	}
	return &appClient{cnc: &cnc}, nil
}
