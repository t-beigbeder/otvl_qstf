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

type istream struct {
	id string
	is quic.Stream
}

var _ IStream = &istream{}

func (is *istream) Id() string {
	return is.id
}

func (is *istream) Read(p []byte) (n int, err error) {
	return is.is.Read(p)
}

type ostream struct {
	id string
	os quic.Stream
}

var _ OStream = &ostream{}

func (os *ostream) Id() string {
	return os.id
}

func (os *ostream) Write(p []byte) (n int, err error) {
	return os.os.Write(p)
}
