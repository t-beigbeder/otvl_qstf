package gst

import (
	"errors"
	"io"
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

func SimpleRoundTrip() {
	h1 := NewHost("host1")
	h2 := NewHost("host2")
	c1, _ := h1.Connect("host2")
	c2 := h2.GetCn("host1")
	s1, _ := c1.OpenStream("simple1")
	s2 := c2.GetStream("simple1")
	_, _, _, _ = c1, c2, s1, s2
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

func StartFuncRoundTrip() {
	h1 := NewHost("host1")
	h2 := NewHost("host2")
	c1, _ := h1.Connect("host2")
	c2 := h2.GetCn("host1")
	s1, _ := c1.OpenStream("simple1")
	s2 := c2.GetStream("simple1")
	_, _, _, _ = c1, c2, s1, s2
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
