package appctl

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
	"log/slog"
	"strconv"
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
	GetCtrlStream() IOStream
	SetSyncStream() error
	GetSyncStream() IOStream
}

type connection struct {
	ctx          context.Context
	qc           quic.Connection
	id           string
	isQuicServer bool
	isAppServer  bool
	ctlStream    IOStream
	syncStream   IOStream
	logger       *slog.Logger
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

func (c *connection) makeQStream(id string, isIn bool) (qst quic.Stream, read int, written int, err error) {
	var (
		accept bool
	)
	if c.isAppServer && c.isQuicServer {
		accept = isIn
	} else if !c.isAppServer && !c.isQuicServer {
		accept = isIn
	} else {
		err = fmt.Errorf(
			"not yet implemented: as %s qs %s in %s",
			strconv.FormatBool(c.isAppServer),
			strconv.FormatBool(c.isQuicServer),
			strconv.FormatBool(isIn))
		return
	}
	if accept {
		if qst, err = c.qc.AcceptStream(c.ctx); err != nil {
			err = fmt.Errorf("error accepting %s stream: %v", id, err)
			return
		}
		_, err = io.ReadFull(qst, make([]byte, 4))
		if err != nil {
			err = fmt.Errorf("error reading %s stream: %v", id, err)
			return
		}
		c.GetLogger().Info("stream accepted", "id", id)
		read = 4
	} else {
		if qst, err = c.qc.OpenStream(); err != nil {
			err = fmt.Errorf("error opening %s stream: %v", id, err)
			return
		}
		_, err = qst.Write(make([]byte, 4))
		if err != nil {
			err = fmt.Errorf("error writing %s stream: %v", id, err)
			return
		}
		c.GetLogger().Info("stream opened", "id", id)
		written = 4
	}
	return
}

func (c *connection) SetCtrlStream() error {
	qst, read, written, err := c.makeQStream("control", c.isAppServer)
	if err != nil {
		return err
	}
	c.ctlStream = NewIOStream("control", qst, read, written)
	return nil
}

func (c *connection) GetCtrlStream() IOStream {
	return c.ctlStream
}

func (c *connection) SetSyncStream() (err error) {
	qst, read, written, err := c.makeQStream("sync", c.isAppServer)
	if err != nil {
		return err
	}
	c.syncStream = NewIOStream("sync", qst, read, written)
	return nil
}

func (c *connection) GetSyncStream() IOStream {
	return c.syncStream
}
