package bfio

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const MaxBufWriterSize = 8192

type BufWriter interface {
	Bytes() []byte
	io.Writer
}

type bufWr struct {
	buf *bytes.Buffer
	out *bufio.Writer
}

var _ BufWriter = &bufWr{}

func (b *bufWr) Write(p []byte) (n int, err error) {
	if len(b.buf.Bytes())+len(p) > MaxBufWriterSize {
		return 0, fmt.Errorf("buffer %d + %d > %d", len(b.buf.Bytes()), len(p), MaxBufWriterSize)
	}
	return b.buf.Write(p)
}

func NewBufWr() BufWriter {
	var buf bytes.Buffer
	return &bufWr{buf: &buf, out: bufio.NewWriter(&buf)}
}

func (b *bufWr) Bytes() []byte {
	return b.buf.Bytes()
}
