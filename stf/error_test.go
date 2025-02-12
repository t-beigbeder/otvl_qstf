package stf

import (
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestErrUnexpectedTerminate(t *testing.T) {
	err := ErrUnexpectedTerminate
	require.Equal(t, ErrUnexpectedTerminate, err)
	require.True(t, ErrUnexpectedTerminate == err)
	err = errors.Join(err, io.ErrClosedPipe)
	require.False(t, ErrUnexpectedTerminate == err)
	err = io.ErrClosedPipe
	require.False(t, ErrUnexpectedTerminate == err)
	err = errors.Join(ErrUnexpectedTerminate, err)
	require.False(t, ErrUnexpectedTerminate == err)
}
