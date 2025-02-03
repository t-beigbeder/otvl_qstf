package fchub

import (
	"github.com/quic-go/quic-go"
)

type IStream interface {
}

type OStream interface{}

type istream struct {
	is quic.Stream
}

var _ IStream = &istream{}

func (is *istream) void() {}
