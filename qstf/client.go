package qstf

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/qstf/quicutils"
	"log/slog"
	"sync"
	"time"
)

type connector struct {
	appProxy *connector
	tcg      *tls.Config
	qcg      *quic.Config
	dto      time.Duration
	addr     string
	id       uuid.UUID
	qcn      quic.Connection
}

type Connector interface {
	HostId() string
	getId() uuid.UUID
}

var _ Connector = &connector{}

func (cnt *connector) HostId() string {
	return cnt.id.String()
}

func (cnt *connector) getId() uuid.UUID {
	return cnt.id
}

func NewConnector(addr, hostId string, qo *quicutils.QuicOptions, dialTimeout time.Duration) (Connector, error) {
	var (
		id  uuid.UUID
		tcg *tls.Config
		qcg *quic.Config
		err error
	)
	if hostId != "" {
		id, err = uuid.Parse(hostId)
	} else {
		id, err = uuid.NewV7()
	}
	if err != nil {
		return nil, err
	}
	if tcg, qcg, err = quicutils.GetConfig(qo); err != nil {
		return nil, err
	}
	cnr := &connector{
		tcg:  tcg,
		qcg:  qcg,
		dto:  dialTimeout,
		addr: addr,
		id:   id,
	}
	return cnr, nil
}

type ClientHost struct {
	ctx            context.Context
	logger         *slog.Logger
	mx             sync.Mutex
	cntRegistry    map[uuid.UUID]*connector
	streamRegistry map[uuid.UUID]*WStream
	HostId         string
}

func NewClientHost(ctx context.Context, logger *slog.Logger, hostId string) *ClientHost {
	ch := &ClientHost{
		ctx:            ctx,
		logger:         logger,
		cntRegistry:    make(map[uuid.UUID]*connector),
		streamRegistry: make(map[uuid.UUID]*WStream),
		HostId:         hostId,
	}
	return ch
}

func (ch *ClientHost) OpenStream(ctx context.Context, cnti Connector) (*WStream, error) {
	var (
		cnt *connector
		qcn quic.Connection
		ok  bool
		err error
	)
	cnt, ok = ch.cntRegistry[cnti.getId()]
	if !ok {
		cnt, ok = cnti.(*connector)
		if !ok {
			return nil, fmt.Errorf("not a connector {%T}", cnti)
		}
		if err = ch.registerCnt(cnt); err != nil {
			return nil, err
		}
	}
	if cnt.qcn == nil {
		qcn, err = netutils.NewQuicConn(cnt.addr, cnt.dto, cnt.tcg, cnt.qcg)
		if err != nil {
			return nil, err
		}
		cnt.qcn = qcn
	}
	return nil, errors.New("not implemented")
}

func (ch *ClientHost) registerCnt(cnt *connector) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.cntRegistry[cnt.id]
	if ok {
		return fmt.Errorf("connector %s already exists", cnt.id)
	}
	ch.cntRegistry[cnt.id] = cnt
	return nil
}

func (ch *ClientHost) unregisterCnt(cnt *connector) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.cntRegistry[cnt.id]
	if !ok {
		return fmt.Errorf("connector %s doesn't exist", cnt.id)
	}
	delete(ch.cntRegistry, cnt.id)
	return nil
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
	err = mst.ch.unregisterStream(mst)
	if err != nil {
		mst.ch.logger.Error("unregisterStream error", "err", err)
		return err
	}
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
