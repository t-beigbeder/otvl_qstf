package qstf

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"log/slog"
	"sync"
)

type baseHost struct {
	ctx            context.Context
	logger         *slog.Logger
	lst            *quic.Listener
	mx             sync.Mutex
	funcRegistry   map[string]func(any) any
	streamRegistry map[string]quic.ReceiveStream
}

// A Host accepts connections from other hosts
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
			streamRegistry: make(map[string]quic.ReceiveStream),
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
		return fmt.Errorf("Function %s already exists", funcName)
	}
	h.bh.funcRegistry[funcName] = f
	return nil
}

func (h *Host) accept() {
	for {
		cnc, err := h.bh.lst.Accept(h.bh.ctx)
		_, _ = cnc, err
	}
}
