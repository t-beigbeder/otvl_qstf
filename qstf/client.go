package qstf

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"log/slog"
	"sync"
	"time"
)

type serverId struct {
	addr   string
	hostId string
}

type ClientHost struct {
	ctx            context.Context
	logger         *slog.Logger
	mx             sync.Mutex
	HostId         string
	serverRegistry map[serverId]quic.Connection
	streamRegistry map[uuid.UUID]*WStream
}

func NewClientHost(ctx context.Context, hostId string) *ClientHost {
	return nil // FIXME
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
