package qstf

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/qstf/quicutils"
	"log/slog"
	"sync"
	"time"
)

type serverId struct {
	addr   string
	hostId string
}

type ServerId interface {
	Addr() string
	HostId() string
}

var _ ServerId = &serverId{}

func NewServerId(addr, hostId string) (ServerId, error) {
	if hostId == "" {
		uuid, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		hostId = uuid.String()
	}
	sid := &serverId{
		addr:   addr,
		hostId: hostId,
	}
	return sid, nil
}

func (s *serverId) Addr() string {
	return s.addr
}

func (s *serverId) HostId() string {
	return s.hostId
}

type connector struct {
	appProxy *connector
	qo       *quicutils.QuicOptions
	addr     string
	id       uuid.UUID
	qc       quic.Connection
}

type ClientHost struct {
	ctx            context.Context
	logger         *slog.Logger
	mx             sync.Mutex
	cnRegistry     map[uuid.UUID]*connector
	streamRegistry map[uuid.UUID]*WStream
	HostId         string
}

func NewClientHost(ctx context.Context, logger *slog.Logger, hostId string) *ClientHost {
	ch := &ClientHost{
		ctx:            ctx,
		logger:         logger,
		cnRegistry:     make(map[uuid.UUID]*connector),
		streamRegistry: make(map[uuid.UUID]*WStream),
		HostId:         hostId,
	}
	return ch
}

func (ch *ClientHost) OpenStream(ctx context.Context, sid ServerId) (*WStream, error) {
	var (
		qc quic.Connection
		ok bool
	)
	if sid.HostId() == "" {
		qc, ok = ch.serverRegistry[sid.HostId()]
		if !ok {
			qc, err := ch.connect()
		}
	}
	return nil, errors.New("not implemented")
}

func (ch *ClientHost) registerStream(mst *WStream) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.streamRegistry[mst.id]
	if ok {
		return fmt.Errorf("stream %s already exists", mst.id)
	}
	ch.streamRegistry[mst.id] = mst
	return nil
}

func (ch *ClientHost) unregisterStream(mst *WStream) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.streamRegistry[mst.id]
	if !ok {
		return fmt.Errorf("stream %s doesn't exist", mst.id)
	}
	delete(ch.streamRegistry, mst.id)
	return nil
}

type WStream struct {
	ch *ClientHost
	ss quic.SendStream
	id uuid.UUID
}

var _ quic.SendStream = &WStream{}

func (mst *WStream) StreamID() quic.StreamID {
	return mst.ss.StreamID()
}

func (mst *WStream) Write(p []byte) (int, error) {
	n, err := mst.ss.Write(p)
	mst.onError(err)
	return n, err
}

func (mst *WStream) Close() error {
	err := mst.ss.Close()
	mst.onError(err)
	return nil
}

func (mst *WStream) CancelWrite(code quic.StreamErrorCode) {
	mst.ss.CancelWrite(code)
}

func (mst *WStream) Context() context.Context {
	return mst.ss.Context()
}

func (mst *WStream) SetWriteDeadline(t time.Time) error {
	return mst.ss.SetWriteDeadline(t)
}

func (mst *WStream) onError(err error) {
	if err == nil {
		return
	}
	iErr := mst.ch.unregisterStream(mst)
	if iErr != nil {
		mst.ch.logger.Error("unregisterStream error", "err", iErr)
	}
}
