package gst

import (
	"errors"
	"fmt"
	_ "github.com/reugn/go-streams"
	_ "github.com/reugn/go-streams/extension"
	ext "github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
	_ "github.com/reugn/go-streams/flow"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/gst"
	"io"
	"slices"
)

var hostsCatalogue map[string]Host

func init() {
	hostsCatalogue = make(map[string]Host)
}

type Host interface {
	Connect(cnId string) (Connection, error)
	GetCn(cnId string) Connection
}

type host struct {
	hostId string
	cns    map[string]Connection
}

var _ Host = (*host)(nil)

func NewHost(hostId string) Host {
	hostsCatalogue[hostId] = &host{hostId: hostId, cns: make(map[string]Connection)}
	return hostsCatalogue[hostId]
}

func (h *host) Connect(hostId string) (Connection, error) {
	h.cns[hostId] = &connection{sourceId: h.hostId, destId: hostId, streams: make(map[string]Stream)}
	th := hostsCatalogue[hostId].(*host)
	th.cns[h.hostId] = &connection{sourceId: hostId, destId: h.hostId, streams: make(map[string]Stream)}
	return h.cns[hostId], nil
}

func (h *host) GetCn(cnId string) Connection {
	return h.cns[cnId]
}

type Connection interface {
	OpenStream(streamId string) (Stream, error)
	GetStream(streamId string) Stream
	AddFunction(fcId string) (Function, error)
}

type connection struct {
	sourceId, destId string
	streams          map[string]Stream
}

func (c *connection) OpenStream(streamId string) (Stream, error) {
	pr, pw := io.Pipe()
	c.streams[streamId] = &stream{wcr: pw}
	dh := hostsCatalogue[c.destId].(*host)
	dc := dh.cns[c.sourceId].(*connection)
	dc.streams[streamId] = &stream{rr: pr, isIn: true}
	return c.streams[streamId], nil
}

func (c *connection) GetStream(streamId string) Stream {
	return c.streams[streamId]
}

func (c *connection) AddFunction(fcId string) (Function, error) {
	return nil, errors.New("not implemented")
}

type Stream interface {
	GetReader() io.Reader
	GetWriter() io.WriteCloser
}

type stream struct {
	isIn bool
	rr   io.Reader
	wcr  io.WriteCloser
}

var _ Stream = (*stream)(nil)

func (s *stream) GetReader() io.Reader {
	return s.rr
}

func (s *stream) GetWriter() io.WriteCloser {
	return s.wcr
}

type Function interface {
}

func setupHosts() (Host, Host, Connection, Connection, Stream, Stream) {
	h1 := NewHost("host1")
	h2 := NewHost("host2")
	c1, _ := h1.Connect("host2")
	c2 := h2.GetCn("host1")
	s1a, _ := c1.OpenStream("simple1a")
	s2a := c2.GetStream("simple1a")
	return h1, h2, c1, c2, s1a, s2a
}

func pocDataSet(label string) []byte {
	return slices.Concat(
		common.Bs2LBs([]byte(fmt.Sprintf("Start %s!", label))),
		common.Bs2LBs([]byte(fmt.Sprintf("Continue %s!", label))),
		common.Bs2LBs([]byte(fmt.Sprintf("Stop %s!", label))),
	)
}

// SimpleRoundTrip writes data from source to stream (s1)
// and reads it in background on other end (s2)
func SimpleRoundTrip() error {
	h1, h2, c1, c2, s1, s2 := setupHosts()
	_, _, _, _, _, _ = h1, h2, c1, c2, s1, s2
	src1 := gst.NewSliceSource(common.Sample("SimpleRoundTrip", true))
	s1Sink, err := gst.NewWriterSink(s1.GetWriter(), gst.LBsWriter)
	if err != nil {
		return err
	}
	ie := errors.New("")
	done := common.BgLaunchErr(func() error {
		src2, err := gst.NewReaderSource(s2.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		sink2 := ext.NewStdoutSink()
		src2.Via(gst.AsStringFlow()).To(sink2)
		sink2.AwaitCompletion()
		return nil
	}, &ie)
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	s1Sink.AwaitCompletion()
	<-done
	if ie != nil {
		return ie
	}
	return nil
}

func LargeRoundTrip() error {
	h1, h2, c1, c2, s1a, s2a := setupHosts()
	_, _, _, _, _, _ = h1, h2, c1, c2, s1a, s2a
	src1 := gst.NewSliceSource(common.LargeSample("LargeRoundTrip"))
	s1Sink, err := gst.NewWriterSink(s1a.GetWriter(), gst.LBsWriter)
	if err != nil {
		return err
	}
	ie := errors.New("")
	done := common.BgLaunchErr(func() error {
		src2, err := gst.NewReaderSource(s2a.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		sink2 := ext.NewFileSink("/dev/null")
		src2.Via(gst.AsStringFlow()).To(sink2)
		sink2.AwaitCompletion()
		return nil
	}, &ie)
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	s1Sink.AwaitCompletion()
	<-done
	if ie != nil {
		return ie
	}
	return nil
}

func TwoReadersRoundTrip() error {
	h1, h2, c1, c2, s1, s2 := setupHosts()
	_, _, _, _, _, _ = h1, h2, c1, c2, s1, s2
	src1 := gst.NewSliceSource(common.Sample("TwoReadersRoundTrip", true))
	s1Sink, err := gst.NewWriterSink(s1.GetWriter(), gst.LBsWriter)
	if err != nil {
		return err
	}
	ie := errors.New("")
	done := common.BgLaunchErr(func() error {
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

		return nil
	}, &ie)
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	s1Sink.AwaitCompletion()
	<-done
	if ie != nil {
		return ie
	}
	return nil
}

func SimuFuncRoundTrip() error {
	h1, h2, c1, c2, s1a, s2a := setupHosts()
	_, _, _, _, _, _ = h1, h2, c1, c2, s1a, s2a
	src1 := gst.NewSliceSource(common.Sample("SimuFuncRoundTrip", true))
	s1Sink, err := gst.NewWriterSink(s1a.GetWriter(), gst.LBsWriter)
	if err != nil {
		return err
	}
	ie := errors.New("")
	done := common.BgLaunchErr(func() error {
		src2, err := gst.NewReaderSource(s2a.GetReader(), gst.LBsReader)
		if err != nil {
			return err
		}
		first := gst.Take(src2, 1)
		first.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return fmt.Sprintf("take header %s", se)
		}, 1)).To(ext.NewStdoutSink())

		s1b, err := c2.OpenStream("simple1b")
		if err != nil {
			return err
		}
		s1bSink, err := gst.NewWriterSink(s1b.GetWriter(), gst.LBsWriter)
		src2.Via(flow.NewMap(func(e any) any {
			se := string(e.([]byte))
			return []byte(fmt.Sprintf("sent back %s", se))
		}, 1)).To(s1bSink)

		return nil
	}, &ie)
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	s2b := c1.GetStream("simple1b")
	src2b, err := gst.NewReaderSource(s2b.GetReader(), gst.LBsReader)
	src2b.Via(flow.NewMap(func(e any) any {
		se := string(e.([]byte))
		return fmt.Sprintf("received back %s", se)
	}, 1)).To(ext.NewStdoutSink())
	<-done
	if ie != nil {
		return ie
	}
	return nil
}
