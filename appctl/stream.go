package appctl

import (
	"github.com/quic-go/quic-go"
	"io"
	"strconv"
)

type IStream interface {
	Id() string
	Qid() string
	io.Reader
}

type OStream interface {
	Id() string
	Qid() string
	io.Writer
}

type IOStream interface {
	Id() string
	Qid() string
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

func (is *istream) Qid() string {
	return strconv.FormatInt(int64(is.is.StreamID()), 10)
}

func (is *istream) Read(p []byte) (n int, err error) {
	n, err = is.is.Read(p)
	is.read += n
	return
}

func NewIStream(id string, is quic.Stream, read int) IStream {
	return &istream{id, is, read}
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

func (os *ostream) Qid() string {
	return strconv.FormatInt(int64(os.os.StreamID()), 10)
}

func (os *ostream) Write(p []byte) (n int, err error) {
	n, err = os.os.Write(p)
	os.written += n
	return
}

func NewOStream(id string, os quic.Stream, written int) OStream {
	return &ostream{id, os, written}
}

type iostream struct {
	istream
	ostream
}

var _ IOStream = &iostream{}

func (ios *iostream) Id() string {
	return ios.istream.id
}

func (ios *iostream) Qid() string {
	return ios.istream.Qid()
}

func NewIOStream(id string, st quic.Stream, read, written int) IOStream {
	return &iostream{
		istream: istream{id, st, read},
		ostream: ostream{id, st, written},
	}
}
