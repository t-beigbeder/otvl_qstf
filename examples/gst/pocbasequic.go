package gst

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"io"
	"log/slog"
	"net"
)

var qhsCatalogue map[string]Host

func init() {
	qhsCatalogue = make(map[string]Host)
}

type qh struct {
	hostId string
	ctx    context.Context
	port   string
	cns    map[string]Connection
	cancel context.CancelFunc
	logger *slog.Logger
}

var _ Host = (*qh)(nil)

func newQcn(ctx context.Context, cId string, qc quic.Connection, logger *slog.Logger) *qcn {
	_, lp, _ := net.SplitHostPort(qc.LocalAddr().String())
	_, rp, _ := net.SplitHostPort(qc.RemoteAddr().String())
	c := &qcn{
		sourceId: lp,
		destId:   rp,
		qc:       qc,
		streams:  make(map[string]Stream),
		logger:   slog.With(logger, "id", cId),
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				stream, err := qc.AcceptStream(ctx)
				if err != nil {
					logger.Error("accepting stream", "err", err)
					return
				}
				logger.Info("accepted stream", "stream", stream.StreamID())
				_ = stream
			}
		}
	}()
	return c
}

func NewQHost(ctx context.Context, hostId string) (Host, error) {
	logger := netutils.GetLoggerFor(hostId)
	port, cancel, err := netutils.RunTestServer(
		"NewQHost", func(ctx context.Context, qc quic.Connection, logger *slog.Logger) {
			logger.Info("NewQHost accept connection", "localAddr", qc.LocalAddr().String())
			h, ok := qhsCatalogue[hostId]
			if !ok {
				logger.Error("NewQHost: host not found", "hostId", hostId)
			}
			dqh := h.(*qh)
			cId := "host2"
			if hostId == "host2" {
				cId = "host1"
			}
			c := newQcn(ctx, cId, qc, logger)
			dqh.cns[cId] = c

		}, logger)
	if err != nil {
		return nil, err
	}
	h := &qh{
		hostId: hostId,
		ctx:    ctx,
		port:   port,
		cns:    make(map[string]Connection),
		cancel: cancel,
		logger: logger,
	}
	qhsCatalogue[hostId] = h
	return h, nil
}

func (h *qh) GetCn(cnId string) Connection {
	return h.cns[cnId]
}

type qcn struct {
	ctx              context.Context
	sourceId, destId string
	qc               quic.Connection
	streams          map[string]Stream
	logger           *slog.Logger
}

var _ Connection = (*qcn)(nil)

func (h *qh) Connect(cnId string) (Connection, error) {
	idqh, ok := qhsCatalogue[cnId]
	if !ok {
		return nil, fmt.Errorf("unknown host id: %s", cnId)
	}
	dqh := idqh.(*qh)
	qc, err := netutils.GetQuicConn("localhost:"+dqh.port, "NewQHost", 0)
	if err != nil {
		return nil, err
	}
	cId := "host2"
	if h.hostId == "host2" {
		cId = "host1"
	}
	c := newQcn(h.ctx, cId, qc, h.logger)
	h.cns[cId] = c
	return c, nil
}

func (q *qcn) OpenStream(streamId string) (Stream, error) {
	wcr, err := q.qc.OpenStream()
	if err != nil {
		return nil, err
	}
	qs := &qst{wcr: wcr}
	q.streams[streamId] = qs
	return qs, nil
}

func (q *qcn) GetStream(streamId string) Stream {
	return q.streams[streamId]
}

type qst struct {
	isIn bool
	rr   io.Reader
	wcr  io.WriteCloser
}

var _ Stream = (*qst)(nil)

func (q *qst) GetReader() io.Reader {
	return q.rr
}

func (q *qst) GetWriter() io.WriteCloser {
	return q.wcr
}

func setupQHosts(ctx context.Context) (Host, Host, Connection, Connection, Stream, Stream) {
	h1, err := NewQHost(ctx, "host1")
	if err != nil {
		panic(err)
	}
	h2, err := NewQHost(ctx, "host2")
	if err != nil {
		panic(err)
	}
	c1, err := h1.Connect("host2")
	if err != nil {
		panic(err)
	}
	s1a, _ := c1.OpenStream("simple1a")

	c2 := h2.GetCn("host1")
	s2a := c1.GetStream("simple1a")

	return h1, h2, c1, c2, s1a, s2a
}
