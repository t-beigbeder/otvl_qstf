package appctl

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
	"log/slog"
	"strconv"
	"sync"
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
	AddIStream(id string) (IStream, error)
	AddOStream(id string) (OStream, error)
	GetIStream(id string) IStream
	GetOStream(id string) OStream
}

type connection struct {
	mmux         sync.Mutex
	ctx          context.Context
	qc           quic.Connection
	id           string
	isQuicServer bool
	isAppServer  bool
	ctlStream    IOStream
	syncStream   IOStream
	iss          map[string]IStream
	oss          map[string]OStream
	logger       *slog.Logger
}

var _ Connection = &connection{}

func NewConnection(ctx context.Context, qc quic.Connection, id string, isQuicServer, isAppServer bool, logger *slog.Logger) Connection {
	return &connection{
		ctx:          ctx,
		qc:           qc,
		id:           id,
		isQuicServer: isQuicServer,
		isAppServer:  isAppServer,
		iss:          make(map[string]IStream),
		oss:          make(map[string]OStream),
		logger:       logger,
	}
}

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

func (c *connection) AddIStream(id string) (IStream, error) {
	if id == "" {
		prefix := "in"
		if !c.isAppServer {
			prefix = "out"
		}
		id = NextId(c.id + "-" + prefix)
	}
	c.mmux.Lock()
	defer c.mmux.Unlock()
	_, ok := c.iss[id]
	if ok {
		return nil, fmt.Errorf("stream already exists: %s", id)
	}
	qst, read, _, err := c.makeQStream(id, true)
	if err != nil {
		return nil, err
	}
	c.iss[id] = NewIStream(id, qst, read)
	return c.iss[id], nil
}

func (c *connection) AddOStream(id string) (OStream, error) {
	if id == "" {
		prefix := "out"
		if !c.isAppServer {
			prefix = "in"
		}
		id = NextId(c.id + "-" + prefix)
	}
	c.mmux.Lock()
	defer c.mmux.Unlock()
	_, ok := c.oss[id]
	if ok {
		return nil, fmt.Errorf("stream already exists: %s", id)
	}
	qst, _, written, err := c.makeQStream(id, false)
	if err != nil {
		return nil, err
	}
	c.oss[id] = NewOStream(id, qst, written)
	return c.oss[id], nil
}

func (c *connection) GetIStream(id string) IStream {
	is, _ := c.iss[id]
	return is
}

func (c *connection) GetOStream(id string) OStream {
	os, _ := c.oss[id]
	return os
}
