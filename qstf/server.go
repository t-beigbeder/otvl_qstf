package qstf

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"io"
	"log/slog"
	"sync"
)

const (
	QstfAlpn = "x-otvl-qstf-v0.1"
)

type serverHost struct {
	logger            *slog.Logger
	lst               *quic.Listener
	mx                sync.Mutex
	funcRegistry      map[string]func(*rStream)
	streamRegistry    map[uuid.UUID]*rStream
	hostId            string
	sdStarted, sdDone chan struct{}
}

// A ServerHost accepts connections from other hosts and registers functions
type ServerHost interface {
	RegisterFunction(funcName string, f func(*rStream)) error
	GetFunction(funcName string) func(*rStream)
	UnregisterFunction(funcName string) error
	HostId() string
	Shutdown()
}

var _ ServerHost = &serverHost{}

// NewServerHost creates host with given hostId to accept QUIC stream openings
// from remote peers on accepted QUIC connections
func NewServerHost(ctx context.Context, logger *slog.Logger, lst *quic.Listener, hostId string) ServerHost {
	sh := &serverHost{
		logger:         logger.With("hostId", hostId),
		lst:            lst,
		funcRegistry:   make(map[string]func(*rStream)),
		streamRegistry: make(map[uuid.UUID]*rStream),
		hostId:         hostId,
		sdStarted:      make(chan struct{}),
		sdDone:         make(chan struct{}),
	}
	go sh.accept(ctx)
	return sh
}

func (sh *serverHost) RegisterFunction(funcName string, f func(*rStream)) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.funcRegistry[funcName]
	if ok {
		return fmt.Errorf("function %s already exists", funcName)
	}
	sh.funcRegistry[funcName] = f
	return nil
}

func (sh *serverHost) GetFunction(funcName string) func(*rStream) {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	fc, _ := sh.funcRegistry[funcName]
	return fc
}

func (sh *serverHost) UnregisterFunction(funcName string) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.funcRegistry[funcName]
	if !ok {
		return fmt.Errorf("function %s doesn't exist", funcName)
	}
	delete(sh.funcRegistry, funcName)
	return nil
}

func (sh *serverHost) HostId() string {
	return sh.hostId
}

func (sh *serverHost) Shutdown() {
	sh.sdStarted <- struct{}{}
	<-sh.sdDone
}

func (sh *serverHost) accept(ctx context.Context) {
	sjc := common.NewJobController(sh.logger)
	go sh.waitForShutdown(sjc)
	for {
		cnc, err := sh.lst.Accept(ctx)
		if err != nil {
			sh.logger.Error("accept error", "err", err)
			break
		}
		cjc := common.NewJobController(sh.logger.With("cnc", cnc.RemoteAddr().String()))
		// FIXME
		err = sh.acceptStreams(ctx, cjc, cnc, false)
		if err != nil {
			break
		}
		err = sh.acceptStreams(ctx, cjc, cnc, true)
		if err != nil {
			break
		}
	}
}

func (sh *serverHost) waitForShutdown(jc common.JobController) {
	for {
		select {
		case <-sh.sdStarted:
			jc.Shutdown()
			sh.sdDone <- struct{}{}
			return
		}
	}
}

func (sh *serverHost) acceptStreams(ctx context.Context, jc common.JobController, cnc quic.Connection, biDir bool) error {
	cnl := cnc.RemoteAddr().String()
	pName := "acceptStreamUni"
	if biDir {
		pName = "acceptStream"
	}
	label := fmt.Sprintf("%s(%s)", pName, cnl)
	cncJob := &common.Job{
		Label: label,
		Run: func(ctx context.Context) error {
			for {
				st, err := cnc.AcceptStream(ctx)
				if err != nil {
					return err
				}
				sh.initStream(st)
			}
		},
	}
	err := jc.RunJob(ctx, cncJob)
	if err != nil {
		return err
	}
	return nil
}

func (sh *serverHost) registerStream(mst *rStream) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.streamRegistry[mst.id]
	if ok {
		return fmt.Errorf("stream %s already exists", mst.id)
	}
	sh.streamRegistry[mst.id] = mst
	return nil
}

func (sh *serverHost) unregisterStream(mst *rStream) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.streamRegistry[mst.id]
	if !ok {
		return fmt.Errorf("stream %s doesn't exist", mst.id)
	}
	delete(sh.streamRegistry, mst.id)
	return nil
}

type Canceler interface {
	Cancel(code quic.StreamErrorCode)
}

type RStream interface {
	io.Reader
	Canceler
}

type rStream struct {
	sh *serverHost
	rs quic.ReceiveStream
	id uuid.UUID
}

var _ RStream = &rStream{}

func (mst *rStream) Read(p []byte) (int, error) {
	n, err := mst.rs.Read(p)
	if err != nil {
		iErr := mst.sh.unregisterStream(mst)
		if iErr != nil {
			mst.sh.logger.Error("unregisterStream error", "err", iErr)
		}
	}
	return n, err
}

func (mst *rStream) Cancel(code quic.StreamErrorCode) {
	mst.rs.CancelRead(code)
	iErr := mst.sh.unregisterStream(mst)
	if iErr != nil {
		mst.sh.logger.Error("unregisterStream error", "err", iErr)
	}
}

func (sh *serverHost) initStream(st quic.ReceiveStream) {
	hid, err := common.LStReader(st)
	if err != nil {
		sh.logger.Error("read hostId error", "err", err)
		return
	}
	_ = hid // FIXME: register peer host
	var stId uuid.UUID
	_, err = io.ReadFull(st, stId[:])
	if err != nil {
		sh.logger.Error("read streamId error", "err", err)
		return
	}
	fn, err := common.LStReader(st)
	if err != nil {
		sh.logger.Error("read funcName error", "err", err)
		return
	}
	sh.logger.Debug("initStream", "hostId", hid, "stId", stId, "fn", fn)
	var fc func(*rStream)
	if fn != "" {
		if fc = sh.GetFunction(fn); fc == nil {
			sh.logger.Error("func does not exist", "funcName", fn)
			return
		}
	}
	mst := &rStream{sh: sh, rs: st, id: stId}
	err = sh.registerStream(mst)
	if err != nil {
		sh.logger.Error("initStream error", "err", err)
		return
	}
	if fn == "" {
		sh.logger.Info("initStream ok", "id", stId)
		return
	}
	sh.logger.Info("initStream ok", "id", stId, "funcName", fn)
	fc(mst)
}
