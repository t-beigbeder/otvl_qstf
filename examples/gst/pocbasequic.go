package gst

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"log/slog"
	"net"
)

var qhsCatalogue map[string]Host

func init() {
	qhsCatalogue = make(map[string]Host)
}

type qh struct {
	hostId string
	port   string
	cns    map[string]Connection
	cancel context.CancelFunc
}

var _ Host = (*qh)(nil)

func NewQHost(hostId string) (Host, error) {
	port, cancel, err := netutils.RunTestServer(
		"NewQHost", func(ctx context.Context, qc quic.Connection, logger *slog.Logger) {
			logger.Info("NewQHost accept connection", "localAddr", qc.LocalAddr().String())
			h, ok := qhsCatalogue[hostId]
			if !ok {
				logger.Error("NewQHost: host not found", "hostId", hostId)
			}
			dqh := h.(*qh)
			_, lp, _ := net.SplitHostPort(qc.LocalAddr().String())
			_, rp, _ := net.SplitHostPort(qc.RemoteAddr().String())
			c := &qcn{
				sourceId: lp,
				destId:   rp,
				qc:       qc,
				streams:  make(map[string]Stream),
			}
			dqh.cns[c.destId] = c

		}, netutils.GetLoggerFor(hostId))
	if err != nil {
		return nil, err
	}
	h := &qh{
		hostId: hostId,
		port:   port,
		cns:    make(map[string]Connection),
		cancel: cancel,
	}
	qhsCatalogue[hostId] = h
	return h, nil
}

func (h *qh) GetCn(cnId string) Connection {
	return h.cns[cnId]
}

type qcn struct {
	sourceId, destId string
	qc               quic.Connection
	streams          map[string]Stream
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
	_, lp, _ := net.SplitHostPort(qc.LocalAddr().String())
	_, rp, _ := net.SplitHostPort(qc.RemoteAddr().String())
	c := &qcn{
		sourceId: lp,
		destId:   rp,
		qc:       qc,
		streams:  make(map[string]Stream),
	}
	h.cns[rp] = c
	return c, nil
}

func (q qcn) OpenStream(streamId string) (Stream, error) {
	//TODO implement me
	panic("implement me")
}

func (q qcn) GetStream(streamId string) Stream {
	//TODO implement me
	panic("implement me")
}

func setupQHosts() (Host, Host, Connection, Connection, Stream, Stream) {
	h1, err := NewQHost("host1")
	if err != nil {
		panic(err)
	}
	h2, err := NewQHost("host2")
	if err != nil {
		panic(err)
	}
	c1, err := h1.Connect("host2")
	if err != nil {
		panic(err)
	}
	c2 := h2.GetCn("host1")

	return h1, h2, c1, c2, nil, nil
}
