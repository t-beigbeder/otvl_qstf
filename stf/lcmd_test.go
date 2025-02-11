package stf

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"testing"
)

func TestNewLocalCommandBasic(t *testing.T) {
	stdin := bytes.NewReader([]byte("hello world"))
	stdout := bfio.NewBufWr()
	stderr := bfio.NewBufWr()
	cs := CommandSpec{
		Cmd: "cat",
	}
	lcfc, err := NewLocalCommand(
		context.Background(),
		cs,
		stdin, stdout, stderr,
		FcId("TestNewLocalCommandBasic"))
	require.NoError(t, err)
	err = lcfc.Run()
	require.NoError(t, err)
	res := stdout.Bytes()
	require.Equal(t, "hello world", string(res))
	res = stderr.Bytes()
	require.Nil(t, res)
}

func TestNewLocalCommandIO(t *testing.T) {
	stdin := bytes.NewReader([]byte("hello world\n"))
	stdout := bfio.NewBufWr()
	stderr := bfio.NewBufWr()
	cs := CommandSpec{
		Cmd:  "grep",
		Args: []string{"o w"},
	}
	lcfc, err := NewLocalCommand(
		context.Background(),
		cs,
		stdin, stdout, stderr,
		FcId("TestNewLocalCommandIO"))
	require.NoError(t, err)
	err = lcfc.Run()
	require.NoError(t, err)
	res := stdout.Bytes()
	require.Equal(t, "hello world\n", string(res))
	res = stderr.Bytes()
	require.Nil(t, res)
}

func TestNewLocalCommandStderr(t *testing.T) {
	stdin := bytes.NewReader(nil)
	stdout := bfio.NewBufWr()
	stderr := bfio.NewBufWr()
	cs := CommandSpec{
		Cmd:  "sh",
		Args: []string{"-c", "echo hello world >&2"},
	}
	lcfc, err := NewLocalCommand(
		context.Background(),
		cs,
		stdin, stdout, stderr,
		FcId("TestNewLocalCommandIO"))
	require.NoError(t, err)
	err = lcfc.Run()
	require.NoError(t, err)
	res := stdout.Bytes()
	require.Nil(t, res)
	res = stderr.Bytes()
	require.Equal(t, "hello world\n", string(res))
}

func TestNewLocalCommandLarge(t *testing.T) {
	instr := ""
	for i := 0; i < 200; i++ {
		instr += fmt.Sprintf("hello world #%d\n", i)
	}
	stdin := bytes.NewReader([]byte(instr))
	stdout := bfio.NewBufWr()
	stderr := bfio.NewBufWr()
	cs := CommandSpec{
		Cmd: "cat",
	}
	lcfc, err := NewLocalCommand(
		context.Background(),
		cs,
		stdin, stdout, stderr,
		FcId("TestNewLocalCommandLarge"))
	require.NoError(t, err)
	err = lcfc.Run()
	require.NoError(t, err)
	res := stdout.Bytes()
	require.Equal(t, instr, string(res))
	res = stderr.Bytes()
	require.Nil(t, res)
}

func TestNewLocalCommandLong(t *testing.T) {
	instr := ""
	for i := 0; i < 100; i++ {
		instr += fmt.Sprintf("hello world #%d\n", i)
	}
	stdin := bytes.NewReader([]byte(instr))
	stdout := bfio.NewBufWr()
	stderr := bfio.NewBufWr()
	cs := CommandSpec{
		Cmd:  "sh",
		Args: []string{"-c", "cat && sleep 0.2"},
	}
	lcfc, err := NewLocalCommand(
		context.Background(),
		cs,
		stdin, stdout, stderr,
		FcId("TestNewLocalCommandLong"))
	require.NoError(t, err)
	err = lcfc.Run()
	require.NoError(t, err)
	res := stdout.Bytes()
	require.Equal(t, instr, string(res))
	res = stderr.Bytes()
	require.Nil(t, res)
}
