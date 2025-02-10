package bfio

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestNewBufWr(t *testing.T) {
	wr := NewBufWr()
	in := bytes.NewReader([]byte("TestNewBufWr"))
	_, err := io.Copy(wr, in)
	require.NoError(t, err)
	require.Equal(t, []byte("TestNewBufWr"), wr.Bytes())
}
