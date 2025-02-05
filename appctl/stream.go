package appctl

import (
	"github.com/quic-go/quic-go"
	"io"
)

type IStream interface {
	Id() string
	io.Reader
}

type OStream interface {
	Id() string
	io.Writer
}

type IOStream interface {
	Id() string
	io.Reader
	io.Writer
}

type istream struct {
	id   string
	is   quic.Stream
	read int
}

var _ IStream = &istream{}

func (is *istream) Id() string {
	return is.id
}

func (is *istream) Read(p []byte) (n int, err error) {
	n, err = is.is.Read(p)
	is.read += n
	return
}

func NewIStream(id string, is quic.Stream) IStream {
	return &istream{id, is, 0}
}

type ostream struct {
	id      string
	os      quic.Stream
	written int
}

var _ OStream = &ostream{}

func (os *ostream) Id() string {
	return os.id
}

func (os *ostream) Write(p []byte) (n int, err error) {
	n, err = os.os.Write(p)
	os.written += n
	return
}

func NewOStream(id string, os quic.Stream) OStream {
	return &ostream{id, os, 0}
}

type iostream struct {
	istream
	ostream
}

var _ IOStream = &iostream{}

func (ios *iostream) Id() string {
	return ios.istream.id
}

func NewIOStream(id string, st quic.Stream, read, written int) IOStream {
	return &iostream{
		istream: istream{id, st, read},
		ostream: ostream{id, st, written},
	}
}
