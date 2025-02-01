package stf

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

type bufwc struct {
	buf *bytes.Buffer
	out *bufio.Writer
}

func (b bufwc) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func newBufwc() *bufwc {
	var buf bytes.Buffer
	return &bufwc{buf: &buf, out: bufio.NewWriter(&buf)}
}

var _ io.Writer = &bufwc{}

func TestNewFuncWrapper(t *testing.T) {
	bs, err := json.Marshal("value for test")
	assert.NoError(t, err)
	wbs := make([]byte, len(bs)+4)
	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
	copy(wbs[4:], bs)
	in := bytes.NewReader(wbs)
	out := newBufwc()
	fcw, err := NewFuncWrapper(context.Background(),
		func(a any) any {
			return "response for " + a.(string)
		},
		in, out)
	assert.NoError(t, err)
	err = fcw.Run()
	assert.NoError(t, err)
	obs := out.buf.Bytes()
	var res any
	json.Unmarshal(obs[4:], &res)
	assert.Equal(t, "response for value for test", res)
}
