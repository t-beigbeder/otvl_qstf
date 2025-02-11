package bfio

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestNewBufWrBase(t *testing.T) {
	wr := NewBufWr()
	in := bytes.NewReader([]byte("TestNewBufWr"))
	_, err := io.Copy(wr, in)
	require.NoError(t, err)
	require.Equal(t, []byte("TestNewBufWr"), wr.Bytes())
}

func TestBufWrExceed(t *testing.T) {
	wr := NewBufWr()
	var err error
	for i := 0; i < 600; i++ {
		in := bytes.NewReader([]byte("TestBufWrExceed"))
		_, err = io.Copy(wr, in)
		if err != nil {
			break
		}
	}
	require.Error(t, err)
}
