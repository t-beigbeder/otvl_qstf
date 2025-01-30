package stf

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

type bufwc struct {
	buf bytes.Buffer
	out *bufio.Writer
}

func (b bufwc) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func (b bufwc) Close() error {
	return nil
}

func newBufwc() *bufwc {
	var buf bytes.Buffer
	return &bufwc{buf: buf, out: bufio.NewWriter(&buf)}
}

var _ io.WriteCloser = &bufwc{}

func TestNewFuncWrapper(t *testing.T) {
	in := strings.NewReader("value for test")
	out := newBufwc()

	fc := NewFuncWrapper(
		context.Background(),
		func(a any) any {
			return fmt.Sprintf("TestNewFuncWrapper: %v", a)
		},
		io.NopCloser(in),
		out,
	)
	err := fc.Start()
	require.NoError(t, err)
	err = fc.Wait()
	require.NoError(t, err)
}
