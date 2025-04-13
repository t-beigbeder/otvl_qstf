package qstf

import (
	"context"
	"crypto/tls"
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
	hostId   string
	uid      uuid.UUID
	qcn      quic.Connection
	logger   *slog.Logger
}

type Connector interface {
	HostId() string
	getUuid() uuid.UUID
}

var _ Connector = &connector{}

func (cnt *connector) HostId() string {
	return cnt.hostId
}

func (cnt *connector) getUuid() uuid.UUID {
	return cnt.uid
}

func (ch *ClientHost) AddConnector(addr, hostId string, qo *quicutils.QuicOptions, dialTimeout time.Duration) (Connector, error) {
	var (
		uid    uuid.UUID
		uidSet bool
		tcg    *tls.Config
		qcg    *quic.Config
		err    error
	)
	if hostId != "" {
		uid, err = uuid.Parse(hostId)
		if err == nil {
			uidSet = true
		}
	}
	if hostId == "" || !uidSet {
		uid, err = uuid.NewV7()
		if err == nil {
			uidSet = true
		} else {
			return nil, err
		}
	}
	if hostId == "" {
		hostId = uid.String()
	}
	if tcg, qcg, err = quicutils.GetConfig(qo); err != nil {
		return nil, err
	}
	cnr := &connector{
		tcg:    tcg,
		qcg:    qcg,
		dto:    dialTimeout,
		addr:   addr,
		hostId: hostId,
		uid:    uid,
		logger: ch.logger.With("addr", addr, "hostId", hostId),
	}
	return cnr, nil
}

type ClientHost struct {
	logger         *slog.Logger
	mx             sync.Mutex
	cntRegistry    map[uuid.UUID]*connector
	streamRegistry map[uuid.UUID]*WStream
	HostId         string
}

func NewClientHost(logger *slog.Logger, hostId string) *ClientHost {
	ch := &ClientHost{
		logger:         logger,
		cntRegistry:    make(map[uuid.UUID]*connector),
		streamRegistry: make(map[uuid.UUID]*WStream),
		HostId:         hostId,
	}
	return ch
}

func (ch *ClientHost) OpenStream(cnti Connector, streamId string) (*WStream, error) {
	var (
		id  uuid.UUID
		cnt *connector
		qcn quic.Connection
		ok  bool
		ss  quic.SendStream
		err error
	)
	if streamId != "" {
		id, err = uuid.Parse(streamId)
	} else {
		id, err = uuid.NewV7()
	}
	if err != nil {
		return nil, err
	}
	cnt, ok = ch.cntRegistry[cnti.getUuid()]
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
		cnt.logger.Info("newQuicConn")
		qcn, err = netutils.NewQuicConn(cnt.addr, cnt.dto, cnt.tcg, cnt.qcg)
		if err != nil {
			cnt.logger.Error("newQuicConn", "err", err)
			return nil, err
		}
		cnt.qcn = qcn
	}
	qcn = cnt.qcn
	if ss, err = qcn.OpenUniStream(); err != nil {
		cnt.logger.Error("openUniStream", "err", err)
		return nil, err
	}
	mst := &WStream{
		ch:     ch,
		qcn:    qcn,
		logger: cnt.logger.With("host", cnt.HostId()),
		ss:     ss,
		id:     id,
	}
	if err = ch.registerStream(mst); err != nil {
		cnt.logger.Error("registerStream", "err", err)
		return nil, err
	}
	return mst, nil
}

func (ch *ClientHost) registerCnt(cnt *connector) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.cntRegistry[cnt.uid]
	if ok {
		return fmt.Errorf("connector %s already exists", cnt.uid)
	}
	ch.cntRegistry[cnt.uid] = cnt
	return nil
}

func (ch *ClientHost) unregisterCnt(cnt *connector) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.cntRegistry[cnt.uid]
	if !ok {
		return fmt.Errorf("connector %s doesn't exist", cnt.uid)
	}
	delete(ch.cntRegistry, cnt.uid)
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
	ch     *ClientHost
	qcn    quic.Connection
	logger *slog.Logger
	ss     quic.SendStream
	id     uuid.UUID
}

var _ quic.SendStream = &WStream{}

func (mst *WStream) StreamID() quic.StreamID {
	return mst.ss.StreamID()
}

func (mst *WStream) Write(p []byte) (int, error) {
	n, err := mst.ss.Write(p)
	mst.onError("write", err)
	return n, err
}

func (mst *WStream) Close() error {
	err := mst.ss.Close()
	mst.onError("close", err)
	iErr := mst.ch.unregisterStream(mst)
	if iErr != nil {
		mst.logger.Error("unregisterStream error", "err", iErr)
	}
	if len(mst.ch.streamRegistry) == 0 {
		sErr := ""
		if err != nil {
			sErr = err.Error()
		}
		iErr = mst.qcn.CloseWithError(0, sErr)
		if iErr != nil {
			mst.ch.logger.Error("closeWithError error", "err", iErr)
		}
	}
	return err
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

func (mst *WStream) onError(from string, err error) {
	if err == nil {
		return
	}
	mst.ch.logger.Error(fmt.Sprintf("%s error", from), "err", err)
	iErr := mst.ch.unregisterStream(mst)
	if iErr != nil {
		mst.logger.Error("unregisterStream error", "err", iErr)
	}
}
