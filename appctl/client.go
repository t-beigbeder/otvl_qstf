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

type AppClient interface {
	AddIStream(id string) (OStream, error)
	GetOStream(id string) (IStream, error)
	RunSyncFunction(id string, in any, out any) error
	NewFunction(id string) error
	FuncAddIStream(fcId string) error
}

type appClient struct {
	cnc    Connection
	ctlMux sync.Mutex
}

var _ AppClient = &appClient{}

func (ac *appClient) reqRoundTrip(cmd string, stId, funcId string, subProcess func(*appClient) error) error {
	ac.ctlMux.Lock()
	defer ac.ctlMux.Unlock()
	crqm := CtrlReqMsg{Command: cmd, StreamId: stId, FunctionId: funcId}
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

func (ac *appClient) RunSyncFunction(funcId string, in any, out any) error {
	err := ac.reqRoundTrip(
		CmdRunSyncFunction,
		"",
		funcId,
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
			if bln > MaxRspSize { // FIXME: depend on funcId
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
	return &appClient{cnc: cnc}, nil
}
