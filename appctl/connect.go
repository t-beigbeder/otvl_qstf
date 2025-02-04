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
	//Connect() (AppClient, error)
}

type connection struct {
	ctx           context.Context
	qc            quic.Connection
	id            string
	isQuicServer  bool
	isAppServer   bool
	ctlStream     quic.Stream
	logger        *slog.Logger
	read, written int
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

func (c *connection) SetCtrlStream() error {
	var err error
	if c.isAppServer && c.isQuicServer {
		if c.ctlStream, err = c.qc.AcceptStream(c.ctx); err != nil {
			return fmt.Errorf("error accepting control stream: %v", err)
		}
		_, err = io.ReadFull(c.ctlStream, make([]byte, 4))
		if err != nil {
			return fmt.Errorf("error reading control stream: %v", err)
		}
		c.GetLogger().Info("Control stream accepted")
	} else if !c.isAppServer && !c.isQuicServer {
		if c.ctlStream, err = c.qc.OpenStream(); err != nil {
			return fmt.Errorf("error opening control stream: %v", err)
		}
		_, err = c.ctlStream.Write(make([]byte, 4))
		if err != nil {
			return fmt.Errorf("error writing control stream: %v", err)
		}
	} else {
		return fmt.Errorf("not yet implemented: %v", *c)
	}
	return nil
}

func (c *connection) GetCtrlStream() quic.Stream {
	return c.ctlStream
}

func (c *connection) AddCtrlRead(i int) {
	c.read += i
}

func (c *connection) AddCtrlWritten(i int) {
	c.written += i
}
