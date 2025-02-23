package gst

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	ext "github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/gst"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"io"
	"log/slog"
	"net"
	"time"
)

var qhsCatalogue map[string]Host

func init() {
	qhsCatalogue = make(map[string]Host)
}

type qh struct {
	hostId string
	ctx    context.Context
	doer   qsDoer
	port   string
	cns    map[string]Connection
	cancel context.CancelFunc
	logger *slog.Logger
}

var _ Host = (*qh)(nil)

func newQcn(h *qh, cId string, qc quic.Connection) *qcn {
	_, lp, _ := net.SplitHostPort(qc.LocalAddr().String())
	_, rp, _ := net.SplitHostPort(qc.RemoteAddr().String())
	c := &qcn{
		h:        h,
		sourceId: lp,
		destId:   rp,
		qc:       qc,
		streams:  make(map[string]Stream),
		logger:   h.logger.With("cId", cId),
	}
	go func() {
		for {
			select {
			case <-h.ctx.Done():
				return
			default:
				c.logger.Debug("accepting stream...")
				rr, err := qc.AcceptStream(h.ctx)
				if err != nil {
					c.logger.Error("accepting stream", "err", err)
					return
				}
				sId := fmt.Sprintf("%08x", rr.StreamID())
				c.logger.Info("accepted stream", "sId", sId)
				if h.doer == nil {
					c.logger.Debug("no doer to execute")
					continue
				}
				qs := &qst{rr: rr, cn: c, h: c.h, logger: c.logger.With("sId", sId)}
				c.streams[sId] = qs
				go func() {
					if err := h.doer(h.ctx, qs); err != nil {
						qs.logger.Error("doing stream", "err", err)
						return
					}
				}()
			}
		}
	}()
	return c
}

type qsDoer func(ctx context.Context, stream Stream) error

func NewQHost(ctx context.Context, hostId string, doer qsDoer) (Host, error) {
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
			c := newQcn(dqh, cId, qc)
			dqh.cns[cId] = c
		}, logger)
	if err != nil {
		return nil, err
	}
	h := &qh{
		hostId: hostId,
		ctx:    ctx,
		port:   port,
		doer:   doer,
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
	h                *qh
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
	c := newQcn(h, cId, qc)
	h.cns[cId] = c
	return c, nil
}

func (q *qcn) OpenStream(streamId string) (Stream, error) {
	wcr, err := q.qc.OpenStream()
	if err != nil {
		return nil, err
	}
	qs := &qst{wcr: wcr, cn: q, h: q.h}
	q.streams[streamId] = qs
	return qs, nil
}

func (q *qcn) GetStream(streamId string) Stream {
	return q.streams[streamId]
}

type qst struct {
	h      *qh
	cn     *qcn
	isIn   bool
	rr     io.Reader
	wcr    io.WriteCloser
	logger *slog.Logger
}

var _ Stream = (*qst)(nil)

func (q *qst) GetReader() io.Reader {
	return q.rr
}

func (q *qst) GetWriter() io.WriteCloser {
	return q.wcr
}

func setupQHosts(ctx context.Context, d1, d2 qsDoer) (Host, Host, error) {
	h1, err := NewQHost(ctx, "host1", d1)
	if err != nil {
		return nil, nil, err
	}
	h2, err := NewQHost(ctx, "host2", d2)
	if err != nil {
		return nil, nil, err
	}
	c1, err := h1.Connect("host2")
	if err != nil {
		return nil, nil, err
	}
	_, err = c1.OpenStream("simple1a")
	if err != nil {
		return nil, nil, err
	}

	return h1, h2, nil
}

// SimpleQuicRoundTrip writes data from source to stream (s1)
// and reads it in background on other end (s2)
func SimpleQuicRoundTrip() error {
	var h2Doer = func(ctx context.Context, s2 Stream) error {
		src2, err := gst.NewReaderSource(s2.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		sink2 := ext.NewStdoutSink()
		src2.Via(gst.AsStringFlow()).To(sink2)
		return err
	}
	h1, h2, err := setupQHosts(context.Background(), nil, h2Doer)
	if err != nil {
		return err
	}
	_ = h2
	src1 := gst.NewSliceSource(common.Sample("SimpleQuicRoundTrip", true))
	c1 := h1.GetCn("host2")
	s1a := c1.GetStream("simple1a")
	s1Sink, err := gst.NewWriterSink(s1a.GetWriter(), gst.LBsWriter)
	if err != nil {
		return err
	}
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	time.Sleep(time.Millisecond * 20)
	return nil
}

func LargeQuicRoundTrip() error {
	var h2Doer = func(ctx context.Context, s2 Stream) error {
		src2, err := gst.NewReaderSource(s2.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		sink2 := ext.NewFileSink("/tmp/LargeQuicRoundTrip.txt")
		src2.Via(gst.AsStringFlow()).To(sink2)
		return err
	}
	h1, h2, err := setupQHosts(context.Background(), nil, h2Doer)
	if err != nil {
		return err
	}
	_ = h2
	src1 := gst.NewSliceSource(common.LargeSample("LargeRoundTrip"))
	c1 := h1.GetCn("host2")
	s1a := c1.GetStream("simple1a")
	s1Sink, err := gst.NewWriterSink(s1a.GetWriter(), gst.LBsWriter)
	if err != nil {
		return err
	}
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	time.Sleep(time.Millisecond * 20)
	return nil
}

func TwoReadersQuicRoundTrip() error {
	var h2Doer = func(ctx context.Context, s2 Stream) error {
		src2, err := gst.NewReaderSource(s2.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}

		twoFirst := gst.Take(src2, 2)
		twoFirst.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return fmt.Sprintf("take2 %s", se)
		}, 1)).To(ext.NewStdoutSink())

		src2.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return fmt.Sprintf("following %s", se)
		}, 1)).To(ext.NewStdoutSink())

		return err
	}
	h1, h2, err := setupQHosts(context.Background(), nil, h2Doer)
	if err != nil {
		return err
	}
	_ = h2
	src1 := gst.NewSliceSource(common.Sample("TwoReadersQuicRoundTrip", true))
	c1 := h1.GetCn("host2")
	s1a := c1.GetStream("simple1a")
	s1Sink, err := gst.NewWriterSink(s1a.GetWriter(), gst.LBsWriter)
	if err != nil {
		return err
	}
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	time.Sleep(time.Millisecond * 20)
	return nil
}

func SimuFuncQuicRoundTrip() error {
	var h2Doer = func(ctx context.Context, s2a Stream) error {
		src2, err := gst.NewReaderSource(s2a.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		first := gst.Take(src2, 1)
		first.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return fmt.Sprintf("take header %s", se)
		}, 1)).To(ext.NewStdoutSink())
		s2ast := s2a.(*qst)
		s1b, err := s2ast.cn.OpenStream("simple1b")
		if err != nil {
			return err
		}
		s1bSink, err := gst.NewWriterSink(s1b.GetWriter(), gst.LBsWriter)
		src2.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return []byte(fmt.Sprintf("sent back %s", se))
		}, 1)).To(s1bSink)

		return nil
	}
	var h1Doer = func(ctx context.Context, s2b Stream) error {
		src2b, err := gst.NewReaderSource(s2b.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		src2b.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return fmt.Sprintf("received back %s", se)
		}, 1)).To(ext.NewStdoutSink())
		return err
	}
	h1, h2, err := setupQHosts(context.Background(), h1Doer, h2Doer)
	if err != nil {
		return err
	}
	_ = h2
	src1 := gst.NewSliceSource(common.Sample("SimuFuncQuicRoundTrip", true))
	c1 := h1.GetCn("host2")
	s1a := c1.GetStream("simple1a")
	s1Sink, err := gst.NewWriterSink(s1a.GetWriter(), gst.LBsWriter)

	src1.Via(flow.NewPassThrough()).To(s1Sink)

	time.Sleep(time.Millisecond * 20)
	return nil
}

func SimuFuncQuicLargeRoundTrip() error {
	var h2Doer = func(ctx context.Context, s2a Stream) error {
		src2, err := gst.NewReaderSource(s2a.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		first := gst.Take(src2, 1)
		first.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return fmt.Sprintf("take header %s", se)
		}, 1)).To(ext.NewStdoutSink())
		s2ast := s2a.(*qst)
		s1b, err := s2ast.cn.OpenStream("simple1b")
		if err != nil {
			return err
		}
		s1bSink, err := gst.NewWriterSink(s1b.GetWriter(), gst.LBsWriter)
		src2.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return []byte(fmt.Sprintf("sent back %s", se))
		}, 1)).To(s1bSink)

		return nil
	}
	var h1Doer = func(ctx context.Context, s2b Stream) error {
		src2b, err := gst.NewReaderSource(s2b.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		src2b.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return fmt.Sprintf("received back %s", se)
		}, 1)).To(ext.NewFileSink("/tmp/SimuFuncQuicLargeRoundTrip.txt"))
		return err
	}
	h1, h2, err := setupQHosts(context.Background(), h1Doer, h2Doer)
	if err != nil {
		return err
	}
	_ = h2
	src1 := gst.NewSliceSource(common.LargeSample("SimuFuncQuicLargeRoundTrip"))
	c1 := h1.GetCn("host2")
	s1a := c1.GetStream("simple1a")
	s1Sink, err := gst.NewWriterSink(s1a.GetWriter(), gst.LBsWriter)

	src1.Via(flow.NewPassThrough()).To(s1Sink)

	time.Sleep(time.Millisecond * 40)
	return nil
}
