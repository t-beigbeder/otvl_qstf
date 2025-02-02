package stf

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewLocalCommandBasic(t *testing.T) {
	stdin := bytes.NewReader([]byte("hello world"))
	stdout := newBufWr()
	stderr := newBufWr()
	cs := CommandSpec{
		Cmd: "cat",
	}
	lcfc, err := NewLocalCommand(
		context.Background(),
		cs,
		stdin, stdout, stderr,
		FcName("TestNewLocalCommandBasic"))
	require.NoError(t, err)
	err = lcfc.Run()
	require.NoError(t, err)
	res, err := fromBufWr(stdout)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(res))
	res, err = fromBufWr(stderr)
	require.NoError(t, err)
	require.Nil(t, res)
}

func TestNewLocalCommandIO(t *testing.T) {
	stdin := bytes.NewReader([]byte("hello world\n"))
	stdout := newBufWr()
	stderr := newBufWr()
	cs := CommandSpec{
		Cmd:  "grep",
		Args: []string{"o w"},
	}
	lcfc, err := NewLocalCommand(
		context.Background(),
		cs,
		stdin, stdout, stderr,
		FcName("TestNewLocalCommandIO"))
	require.NoError(t, err)
	err = lcfc.Run()
	require.NoError(t, err)
	res, err := fromBufWr(stdout)
	require.NoError(t, err)
	require.Equal(t, "hello world\n", string(res))
	res, err = fromBufWr(stderr)
	require.NoError(t, err)
	require.Nil(t, res)
}
