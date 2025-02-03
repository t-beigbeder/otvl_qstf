package appctl

import "github.com/quic-go/quic-go"

type Connection interface {
	GetId() string
	GetQuicConnection() quic.Connection
	//Connect() (AppClient, error)
}

type connection struct {
	qc       quic.Connection
	id       string
	isServer bool
	iCtl     istream
	oCtl     ostream
}

var _ Connection = &connection{}

func (c *connection) GetId() string {
	return c.id
}

func (c *connection) GetQuicConnection() quic.Connection {
	return c.qc
}
