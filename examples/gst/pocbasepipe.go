package gst

import (
	_ "github.com/reugn/go-streams"
	_ "github.com/reugn/go-streams/extension"
	_ "github.com/reugn/go-streams/flow"
	"io"
)

var pphsCatalogue map[string]Host

func init() {
	pphsCatalogue = make(map[string]Host)
}

type pph struct {
	hostId string
	cns    map[string]Connection
}

var _ Host = (*pph)(nil)

func NewPPHost(hostId string) Host {
	pphsCatalogue[hostId] = &pph{hostId: hostId, cns: make(map[string]Connection)}
	return pphsCatalogue[hostId]
}

func (h *pph) Connect(hostId string) (Connection, error) {
	h.cns[hostId] = &ppcn{sourceId: h.hostId, destId: hostId, streams: make(map[string]Stream)}
	th := pphsCatalogue[hostId].(*pph)
	th.cns[h.hostId] = &ppcn{sourceId: hostId, destId: h.hostId, streams: make(map[string]Stream)}
	return h.cns[hostId], nil
}

func (h *pph) GetCn(cnId string) Connection {
	return h.cns[cnId]
}

type ppcn struct {
	sourceId, destId string
	streams          map[string]Stream
}

func (c *ppcn) OpenStream(streamId string) (Stream, error) {
	pr, pw := io.Pipe()
	c.streams[streamId] = &ppst{wcr: pw}
	dh := pphsCatalogue[c.destId].(*pph)
	dc := dh.cns[c.sourceId].(*ppcn)
	dc.streams[streamId] = &ppst{rr: pr, isIn: true}
	return c.streams[streamId], nil
}

func (c *ppcn) GetStream(streamId string) Stream {
	return c.streams[streamId]
}

type ppst struct {
	isIn bool
	rr   io.Reader
	wcr  io.WriteCloser
}

var _ Stream = (*ppst)(nil)

func (s *ppst) GetReader() io.Reader {
	return s.rr
}

func (s *ppst) GetWriter() io.WriteCloser {
	return s.wcr
}

func setupPPHosts() (Host, Host, Connection, Connection, Stream, Stream) {
	h1 := NewPPHost("host1")
	h2 := NewPPHost("host2")
	c1, _ := h1.Connect("host2")
	c2 := h2.GetCn("host1")
	s1a, _ := c1.OpenStream("simple1a")
	s2a := c2.GetStream("simple1a")
	return h1, h2, c1, c2, s1a, s2a
}
