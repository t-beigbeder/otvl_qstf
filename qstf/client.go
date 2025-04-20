package qstf

import (
	"crypto/tls"
	"fmt"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/qstf/quicutils"
	"io"
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
	bidir    bool
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

func NewConnector(addr, hostId string, qo *quicutils.QuicOptions, dialTimeout time.Duration, logger *slog.Logger) (Connector, error) {
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
		bidir:  qo.BidirStream,
		logger: logger.With("addr", addr, "hostId", hostId),
	}
	return cnr, nil
}

type clientHost struct {
	logger         *slog.Logger
	mx             sync.Mutex
	cntRegistry    map[uuid.UUID]*connector
	streamRegistry map[uuid.UUID]*wStream
	hostId         string
	jc             common.JobController
}

func (ch *clientHost) RegisterFunction(funcName string, f func(*rStream)) error {
	//TODO implement me
	panic("implement me")
}

func (ch *clientHost) GetFunction(funcName string) func(*rStream) {
	//TODO implement me
	panic("implement me")
}

func (ch *clientHost) UnregisterFunction(funcName string) error {
	//TODO implement me
	panic("implement me")
}

func (ch *clientHost) Shutdown() {
	//TODO implement me
	panic("implement me")
}

type ClientHost interface {
	OpenStream(cnti Connector, streamId string, fName string) (WStream, error)
	ServerHost
}

var _ ClientHost = &clientHost{}

func NewClientHost(logger *slog.Logger, hostId string) ClientHost {
	ch := &clientHost{
		logger:         logger,
		cntRegistry:    make(map[uuid.UUID]*connector),
		streamRegistry: make(map[uuid.UUID]*wStream),
		hostId:         hostId,
		jc:             common.NewJobController(logger),
	}
	return ch
}

func (ch *clientHost) HostId() string { return ch.hostId }

func (ch *clientHost) OpenStream(cnti Connector, streamId string, fName string) (WStream, error) {
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
	if cnt.bidir {
		ss, err = qcn.OpenStream()
	} else {
		ss, err = qcn.OpenUniStream()
	}
	if err != nil {
		cnt.logger.Error("openUniStream", "err", err)
		return nil, err
	}
	if err = common.LstWriter(ss, cnt.hostId); err != nil {
		cnt.logger.Error("openStream: lstWriter hostId", "err", err)
		return nil, err
	}
	if _, err = ss.Write(id[:]); err != nil {
		cnt.logger.Error("openStream: lstWriter streamId", "err", err)
		return nil, err
	}
	if err = common.LstWriter(ss, fName); err != nil {
		cnt.logger.Error("openStream: lstWriter fName", "err", err)
		return nil, err
	}
	mst := &wStream{
		ch:     ch,
		qcn:    qcn,
		logger: cnt.logger.With("host", cnt.HostId(), "id", id, "fName", fName),
		ss:     ss,
		id:     id,
		done:   make(chan struct{}),
	}
	if err = ch.registerStream(mst); err != nil {
		cnt.logger.Error("registerStream", "err", err)
		return nil, err
	}
	return mst, nil
}

func (ch *clientHost) registerCnt(cnt *connector) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.cntRegistry[cnt.uid]
	if ok {
		return fmt.Errorf("connector %s already exists", cnt.uid)
	}
	ch.cntRegistry[cnt.uid] = cnt
	return nil
}

func (ch *clientHost) unregisterCnt(cnt *connector) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.cntRegistry[cnt.uid]
	if !ok {
		return fmt.Errorf("connector %s doesn't exist", cnt.uid)
	}
	delete(ch.cntRegistry, cnt.uid)
	return nil
}

func (ch *clientHost) registerStream(mst *wStream) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.streamRegistry[mst.id]
	if ok {
		return fmt.Errorf("stream %s already exists", mst.id)
	}
	ch.streamRegistry[mst.id] = mst
	return nil
}

func (ch *clientHost) unregisterStream(mst *wStream) error {
	ch.mx.Lock()
	defer ch.mx.Unlock()
	_, ok := ch.streamRegistry[mst.id]
	if !ok {
		return fmt.Errorf("stream %s doesn't exist", mst.id)
	}
	delete(ch.streamRegistry, mst.id)
	return nil
}

type wStream struct {
	ch     *clientHost
	qcn    quic.Connection
	logger *slog.Logger
	mx     sync.Mutex
	closed bool
	ss     quic.SendStream
	id     uuid.UUID
	done   chan struct{}
}

type WStream interface {
	io.WriteCloser
	Canceler
}

func (mst *wStream) Write(p []byte) (int, error) {
	n, err := mst.ss.Write(p)
	mst.onError("write", err)
	return n, err
}

func (mst *wStream) close(cancel bool) error {
	mst.mx.Lock()
	defer mst.mx.Unlock()
	var err error
	if !mst.closed && !cancel {
		mst.logger.Debug("close")
		err = mst.ss.Close()
		mst.onError("close", err)
	}
	mst.closed = true
	iErr := mst.ch.unregisterStream(mst)
	if iErr != nil {
		mst.logger.Error("unregisterStream error", "err", iErr)
	}
	if len(mst.ch.streamRegistry) == 0 {
		sErr := ""
		if err != nil {
			sErr = err.Error()
		}
		mst.logger.Info("close QUIC connection", "err", sErr)
		iErr = mst.qcn.CloseWithError(0, sErr)
		if iErr != nil {
			mst.ch.logger.Error("closeWithError error", "err", iErr)
		}
	}
	return err
}

func (mst *wStream) Close() error {
	return mst.close(false)
}

func (mst *wStream) Cancel(code quic.StreamErrorCode) {
	mst.ss.CancelWrite(code)
	_ = mst.close(true)
}

func (mst *wStream) onError(from string, err error) {
	if err == nil {
		return
	}
	mst.ch.logger.Error(fmt.Sprintf("%s error", from), "err", err)
	iErr := mst.ch.unregisterStream(mst)
	if iErr != nil {
		mst.logger.Error("unregisterStream error", "err", iErr)
	}
}
