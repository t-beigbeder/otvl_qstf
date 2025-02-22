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
	s1, _ := c1.OpenStream("simple1")
	s2 := c2.GetStream("simple1")
	return h1, h2, c1, c2, s1, s2
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
	var intErr error
	done := common.BgLaunch(func() {
		src2, err := gst.NewReaderSource(s2.GetReader(), gst.LBsReader)
		if err != nil {
			intErr = err
			return
		}
		sink2 := ext.NewStdoutSink()
		src2.Via(gst.AsStringFlow()).To(sink2)
		sink2.AwaitCompletion()
	})
	src1.Via(flow.NewPassThrough()).To(s1Sink)
	s1Sink.AwaitCompletion()
	<-done
	return intErr
}

func StartFuncRoundTrip() {
	h1, h2, c1, c2, s1, s2 := setupHosts()
	_, _, _, _, _, _ = h1, h2, c1, c2, s1, s2
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			bs := make([]byte, 1024)
			n, err := s2.GetReader().Read(bs)
			if err != nil && err != io.EOF {
				break
			}
			println(string(bs[:n]))
			if err != nil {
				break
			}
		}
	}()
	s1.GetWriter().Write([]byte("hello"))
	s1.GetWriter().Close()
	<-done
}
