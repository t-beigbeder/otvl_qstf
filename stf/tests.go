package stf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

func toJsonBytes(a any) ([]byte, error) {
	bs, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	wbs := make([]byte, len(bs)+4)
	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
	copy(wbs[4:], bs)
	return wbs, err
}

type bufWr struct {
	buf *bytes.Buffer
	out *bufio.Writer
}

var _ io.Writer = &bufWr{}

func (b *bufWr) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func newBufWr() io.Writer {
	var buf bytes.Buffer
	return &bufWr{buf: &buf, out: bufio.NewWriter(&buf)}
}

func fromJsonBytes(wr io.Writer) (any, error) {
	o, ok := wr.(*bufWr)
	if !ok {
		return nil, fmt.Errorf("expected bufWr got %T", wr)
	}
	obs := o.buf.Bytes()
	bln := binary.BigEndian.Uint32(obs[0:4])
	if bln != uint32(len(obs)-4) {
		return nil, fmt.Errorf("invalid number of bytes read: expected %d got %d", len(obs)-4, bln)
	}
	var a any
	err := json.Unmarshal(obs[4:], &a)
	if err != nil {
		return nil, err
	}
	return a, nil
}
