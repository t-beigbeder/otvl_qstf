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
	"time"
)

const (
	QstfAlpn = "x-otvl-qstf-v0.1"
)

// A ServerHost accepts connections from other hosts and registers functions
type ServerHost struct {
	ctx            context.Context
	logger         *slog.Logger
	lst            *quic.Listener
	mx             sync.Mutex
	funcRegistry   map[string]func(any) any
	streamRegistry map[uuid.UUID]*Stream
	HostId         string
}

// NewServerHost creates host with given hostId to accept connections
func NewServerHost(ctx context.Context, logger *slog.Logger, lst *quic.Listener, hostId string) *ServerHost {
	sh := &ServerHost{
		ctx:            ctx,
		logger:         logger.With("hostId", hostId),
		lst:            lst,
		funcRegistry:   make(map[string]func(any) any),
		streamRegistry: make(map[uuid.UUID]*Stream),
		HostId:         hostId,
	}
	go sh.accept()
	return sh
}

func (sh *ServerHost) RegisterFunction(funcName string, f func(any) any) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.funcRegistry[funcName]
	if ok {
		return fmt.Errorf("function %s already exists", funcName)
	}
	sh.funcRegistry[funcName] = f
	return nil
}

func (sh *ServerHost) GetFunction(funcName string) func(any) any {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	fc, _ := sh.funcRegistry[funcName]
	return fc
}

func (sh *ServerHost) UnregisterFunction(funcName string) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.funcRegistry[funcName]
	if !ok {
		return fmt.Errorf("function %s doesn't exist", funcName)
	}
	delete(sh.funcRegistry, funcName)
	return nil
}

func (sh *ServerHost) accept() {
	for {
		cnc, err := sh.lst.Accept(sh.ctx)
		if err != nil {
			sh.logger.Error("accept error", "err", err)
			return
		}
		go func() {
			for {
				st, err := cnc.AcceptStream(sh.ctx)
				if err != nil {
					sh.logger.Error("accept stream error", "err", err)
					return
				}
				sh.initStream(st)
			}
		}()
		go func() {
			for {
				st, err := cnc.AcceptUniStream(sh.ctx)
				if err != nil {
					sh.logger.Error("accept unistream error", "err", err)
					return
				}
				sh.initStream(st)
			}
		}()
	}
}

type Stream struct {
	sh *ServerHost
	rs quic.ReceiveStream
	id uuid.UUID
}

var _ quic.ReceiveStream = &Stream{}

func (mst *Stream) StreamID() quic.StreamID {
	return mst.rs.StreamID()
}

func (mst *Stream) Read(p []byte) (int, error) {
	n, err := mst.rs.Read(p)
	if err != nil {
		iErr := mst.sh.unregisterStream(mst)
		if iErr != nil {
			mst.sh.logger.Error("unregisterStream error", "err", iErr)
		}
	}
	return n, err
}

func (mst *Stream) CancelRead(code quic.StreamErrorCode) {
	mst.rs.CancelRead(code)
	iErr := mst.sh.unregisterStream(mst)
	if iErr != nil {
		mst.sh.logger.Error("unregisterStream error", "err", iErr)
	}
}

func (mst *Stream) SetReadDeadline(t time.Time) error {
	return mst.rs.SetReadDeadline(t)
}

func (sh *ServerHost) initStream(st quic.ReceiveStream) {
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
	if fn != "" && sh.GetFunction(fn) == nil {
		sh.logger.Error("func does not exist", "funcName", fn)
		return
	}
	mst := &Stream{sh: sh, rs: st, id: stId}
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
	// FIXME: initialize a pipeline and registers the flow
}

func (sh *ServerHost) registerStream(mst *Stream) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.streamRegistry[mst.id]
	if ok {
		return fmt.Errorf("stream %s already exists", mst.id)
	}
	sh.streamRegistry[mst.id] = mst
	return nil
}

func (sh *ServerHost) unregisterStream(mst *Stream) error {
	sh.mx.Lock()
	defer sh.mx.Unlock()
	_, ok := sh.streamRegistry[mst.id]
	if !ok {
		return fmt.Errorf("stream %s doesn't exist", mst.id)
	}
	delete(sh.streamRegistry, mst.id)
	return nil
}
