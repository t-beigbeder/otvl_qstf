package appctl

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
	"log/slog"
)

const (
	QstfAlpn = "x-otvl-qstf-v0.1"
)

type Connection interface {
	GetCtx() context.Context
	GetId() string
	GetQuicConnection() quic.Connection
	GetLogger() *slog.Logger
	SetCtrlStream() error
	GetCtrlStream() quic.Stream
	AddCtrlRead(int)
	AddCtrlWritten(int)
	SetSyncStream() error
	GetSyncStream() quic.Stream
	AddSyncRead(int)
	AddSyncWritten(int)
	//Connect() (AppClient, error)
}

type connection struct {
	ctx                   context.Context
	qc                    quic.Connection
	id                    string
	isQuicServer          bool
	isAppServer           bool
	ctlStream             quic.Stream
	ctlRead, ctlWritten   int
	syncStream            quic.Stream
	syncRead, syncWritten int
	logger                *slog.Logger
}

var _ Connection = &connection{}

func (c *connection) GetCtx() context.Context {
	return c.ctx
}

func (c *connection) GetId() string {
	return c.id
}

func (c *connection) GetQuicConnection() quic.Connection {
	return c.qc
}

func (c *connection) GetLogger() *slog.Logger {
	return c.logger
}
func (c *connection) setIoStream(isSync bool) (iost quic.Stream, err error) {
	stn := "control"
	if isSync {
		stn = "sync"
	}
	if c.isAppServer && c.isQuicServer {
		if iost, err = c.qc.AcceptStream(c.ctx); err != nil {
			err = fmt.Errorf("error accepting %s stream: %v", stn, err)
			return
		}
		_, err = io.ReadFull(iost, make([]byte, 4))
		if err != nil {
			err = fmt.Errorf("error reading %s stream: %v", stn, err)
			return
		}
		if !isSync {
			c.GetLogger().Info("Control stream accepted")
		} else {
			c.GetLogger().Info("Sync stream accepted")
		}
	} else if !c.isAppServer && !c.isQuicServer {
		if iost, err = c.qc.OpenStream(); err != nil {
			err = fmt.Errorf("error opening %s stream: %v", stn, err)
			return
		}
		_, err = iost.Write(make([]byte, 4))
		if err != nil {
			err = fmt.Errorf("error writing %s stream: %v", stn, err)
			return
		}
	} else {
		err = fmt.Errorf("not yet implemented: %v", *c)
		return
	}
	return iost, nil
}

func (c *connection) SetCtrlStream() (err error) {
	c.ctlStream, err = c.setIoStream(false)
	return
}

func (c *connection) GetCtrlStream() quic.Stream {
	return c.ctlStream
}

func (c *connection) AddCtrlRead(i int) {
	c.ctlRead += i
}

func (c *connection) AddCtrlWritten(i int) {
	c.ctlWritten += i
}

func (c *connection) SetSyncStream() (err error) {
	c.syncStream, err = c.setIoStream(true)
	return
}

func (c *connection) GetSyncStream() quic.Stream {
	return c.syncStream
}

func (c *connection) AddSyncRead(i int) {
	c.syncRead += i
}

func (c *connection) AddSyncWritten(i int) {
	c.syncWritten += i
}
