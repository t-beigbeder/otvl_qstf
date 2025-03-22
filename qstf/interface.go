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

type baseHost struct {
	ctx            context.Context
	logger         *slog.Logger
	lst            *quic.Listener
	mx             sync.Mutex
	funcRegistry   map[string]func(any) any
	streamRegistry map[uuid.UUID]Stream
}

// A Host accepts connections from other hosts and registers functions
type Host struct {
	bh     *baseHost
	HostId string
}

// NewHost creates host with given hostId to accept connections
func NewHost(ctx context.Context, logger *slog.Logger, lst *quic.Listener, hostId string) *Host {
	h := &Host{
		bh: &baseHost{
			ctx:            ctx,
			logger:         logger.With("hostId", hostId),
			lst:            lst,
			funcRegistry:   make(map[string]func(any) any),
			streamRegistry: make(map[uuid.UUID]Stream),
		},
		HostId: hostId,
	}
	go h.accept()
	return h
}

func (h *Host) RegisterFunction(funcName string, f func(any) any) error {
	h.bh.mx.Lock()
	defer h.bh.mx.Unlock()
	_, ok := h.bh.funcRegistry[funcName]
	if ok {
		return fmt.Errorf("function %s already exists", funcName)
	}
	h.bh.funcRegistry[funcName] = f
	return nil
}

func (h *Host) GetFunction(funcName string) func(any) any {
	h.bh.mx.Lock()
	defer h.bh.mx.Unlock()
	fc, _ := h.bh.funcRegistry[funcName]
	return fc
}

func (h *Host) UnregisterFunction(funcName string) error {
	h.bh.mx.Lock()
	defer h.bh.mx.Unlock()
	_, ok := h.bh.funcRegistry[funcName]
	if !ok {
		return fmt.Errorf("function %s doesn't exist", funcName)
	}
	delete(h.bh.funcRegistry, funcName)
	return nil
}

func (h *Host) accept() {
	for {
		cnc, err := h.bh.lst.Accept(h.bh.ctx)
		if err != nil {
			h.bh.logger.Error("accept error", "err", err)
			return
		}
		go func() {
			for {
				h.initStream(cnc.AcceptStream(h.bh.ctx))
			}
		}()
		go func() {
			for {
				h.initStream(cnc.AcceptUniStream(h.bh.ctx))
			}
		}()
	}
}

type Stream struct {
	h  *Host
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
		iErr := mst.h.unregisterStream(mst)
		if iErr != nil {
			mst.h.bh.logger.Error("unregisterStream error", "err", iErr)
		}
	}
	return n, err
}

func (mst *Stream) CancelRead(code quic.StreamErrorCode) {
	mst.rs.CancelRead(code)
	iErr := mst.h.unregisterStream(mst)
	if iErr != nil {
		mst.h.bh.logger.Error("unregisterStream error", "err", iErr)
	}
}

func (mst *Stream) SetReadDeadline(t time.Time) error {
	return mst.rs.SetReadDeadline(t)
}

func (h *Host) initStream(st quic.ReceiveStream, err error) {
	if err != nil {
		h.bh.logger.Error("accept Stream error", "err", err)
		return
	}
	var stId uuid.UUID
	_, err = io.ReadFull(st, stId[:])
	if err != nil {
		h.bh.logger.Error("read streamId error", "err", err)
		return
	}
	fn, err := common.LStReader(st)
	if err != nil {
		h.bh.logger.Error("read funcName error", "err", err)
		return
	}
	if fn != "" && h.GetFunction(fn) == nil {
		h.bh.logger.Error("func does not exist", "funcName", fn)
		return
	}
	mst := &Stream{h: h, rs: st, id: stId}
	err = h.registerStream(mst)
	if err != nil {
		h.bh.logger.Error("initStream error", "err", err)
		return
	}
	if fn == "" {
		h.bh.logger.Info("initStream ok", "id", stId)
		return
	}
	h.bh.logger.Info("initStream ok", "id", stId, "funcName", fn)
	// FIXME: initialize a pipeline and registers the flow
}

func (h *Host) registerStream(mst *Stream) error {
	h.bh.mx.Lock()
	defer h.bh.mx.Unlock()
	_, ok := h.bh.streamRegistry[mst.id]
	if ok {
		return fmt.Errorf("stream %s already exists", mst.id)
	}
	return nil
}

func (h *Host) unregisterStream(mst *Stream) error {
	h.bh.mx.Lock()
	defer h.bh.mx.Unlock()
	_, ok := h.bh.streamRegistry[mst.id]
	if !ok {
		return fmt.Errorf("stream %s doesn't exist", mst.id)
	}
	delete(h.bh.streamRegistry, mst.id)
	return nil
}
