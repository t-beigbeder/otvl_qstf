package qstf

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewServerHost(t *testing.T) {
	td := t.TempDir()
	sh, port, cancel, err := RunQstfTestServerWithCtc(td, nil)
	require.NoError(t, err)
	require.NotNil(t, sh)
	require.NotNil(t, port)
	time.Sleep(time.Millisecond * 10)
	cancel()
}
