package appctl

import (
	"github.com/quic-go/quic-go"
	"io"
	"log/slog"
	"strconv"
)

type IStream interface {
	Id() string
	Qid() string
	QStream() quic.Stream
	io.Reader
}

type OStream interface {
	Id() string
	Qid() string
	QStream() quic.Stream
	io.Writer
}

type IOStream interface {
	Id() string
	Qid() string
	QStream() quic.Stream
	io.Reader
	io.Writer
}

type istream struct {
	logger *slog.Logger
	id     string
	is     quic.Stream
	read   int
}

var _ IStream = &istream{}

func (is *istream) Id() string {
	return is.id
}

func (is *istream) Qid() string {
	return strconv.FormatInt(int64(is.is.StreamID()), 10)
}

func (is *istream) Read(p []byte) (n int, err error) {
	is.logger.Debug("reading data", "lenp", len(p))
	n, err = is.is.Read(p)
	is.logger.Debug("read data", "n", n, "err", err)
	is.read += n
	return
}

func (is *istream) QStream() quic.Stream {
	return is.is
}

func NewIStream(logger *slog.Logger, id string, is quic.Stream, read int) IStream {
	return &istream{logger, id, is, read}
}

type ostream struct {
	logger  *slog.Logger
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
	os.logger.Debug("writing data", "lenp", len(p))
	n, err = os.os.Write(p)
	os.logger.Debug("written data", "n", n, "err", err)
	os.written += n
	return
}

func (os *ostream) QStream() quic.Stream {
	return os.os
}

func NewOStream(logger *slog.Logger, id string, os quic.Stream, written int) OStream {
	return &ostream{logger, id, os, written}
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

func (ios *iostream) QStream() quic.Stream {
	return ios.istream.is
}

func NewIOStream(logger *slog.Logger, id string, st quic.Stream, read, written int) IOStream {
	return &iostream{
		istream: istream{logger, id, st, read},
		ostream: ostream{logger, id, st, written},
	}
}
