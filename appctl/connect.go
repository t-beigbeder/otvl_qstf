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
	String() string
	GetLogger() *slog.Logger
	SetCtrlStream() error
	GetCtrlStream() IOStream
	AddIStream(id string) (IStream, error)
	AddOStream(id string) (OStream, error)
	GetIStream(id string) IStream
	GetOStream(id string) OStream
	GetIStreams() map[string]IStream
	GetOStreams() map[string]OStream
}

type connection struct {
	mux          sync.Mutex
	ctx          context.Context
	qc           quic.Connection
	id           string
	isQuicServer bool
	isAppServer  bool
	ctlStream    IOStream
	iss          map[string]IStream
	oss          map[string]OStream
	logger       *slog.Logger
}

var _ Connection = &connection{}

func NewConnection(ctx context.Context, qc quic.Connection, id string, isQuicServer, isAppServer bool, logger *slog.Logger) Connection {
	cn := &connection{
		qc:           qc,
		id:           id,
		isQuicServer: isQuicServer,
		isAppServer:  isAppServer,
		iss:          make(map[string]IStream),
		oss:          make(map[string]OStream),
		logger:       logger.With("connId", id),
	}
	cn.ctx = context.WithValue(ctx, "cn", cn)
	return cn
}

func (c *connection) GetCtx() context.Context {
	return c.ctx
}

func (c *connection) GetId() string {
	return c.id
}

func (c *connection) String() string {
	return fmt.Sprintf("%s->%s", c.qc.LocalAddr(), c.qc.RemoteAddr())
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
	c.logger.Debug("set ctrl stream", "id", c.id, "qid", c.ctlStream.Qid())
	return nil
}

func (c *connection) GetCtrlStream() IOStream {
	return c.ctlStream
}

func (c *connection) AddIStream(id string) (IStream, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	_, ok := c.iss[id]
	if ok {
		return nil, fmt.Errorf("istream already exists: %s", id)
	}
	qst, read, _, err := c.makeQStream(id, true)
	if err != nil {
		return nil, err
	}
	c.iss[id] = NewIStream(id, qst, read)
	c.logger.Debug("new istream created", "id", id, "qid", c.iss[id].Qid())
	return c.iss[id], nil
}

func (c *connection) AddOStream(id string) (OStream, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	_, ok := c.oss[id]
	if ok {
		return nil, fmt.Errorf("ostream already exists: %s", id)
	}
	qst, _, written, err := c.makeQStream(id, false)
	if err != nil {
		return nil, err
	}
	c.oss[id] = NewOStream(id, qst, written)
	c.logger.Debug("new ostream created", "id", id, "qid", c.oss[id].Qid())
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

func (c *connection) GetIStreams() map[string]IStream {
	return c.iss
}

func (c *connection) GetOStreams() map[string]OStream {
	return c.oss
}
