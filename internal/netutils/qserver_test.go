package netutils

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestGetQuicListenerBasic(t *testing.T) {
	cert, err := SelfSigned("localhost")
	require.NoError(t, err)
	require.NotNil(t, cert)
	tc, qc := GetQuicConfigFor(cert, "TestGetQuicListenerBasic", nil)
	listener, host, port, err := GetQuicListener(":0", tc, qc)
	require.NoError(t, err)
	require.NotNil(t, listener)
	as := listener.Addr().String()
	require.True(t, strings.HasPrefix(as, "0.0.0.0:"))
	require.Equal(t, "0.0.0.0", host)
	require.Equal(t, as, fmt.Sprintf("%s:%s", host, port))
}

func TestGetQuicListenerLocal(t *testing.T) {
	cert, err := SelfSigned("localhost")
	require.NoError(t, err)
	tc, qc := GetQuicConfigFor(cert, "TestGetQuicListenerLocal", nil)
	listener, host, port, err := GetQuicListener("127.0.0.1:0", tc, qc)
	require.NoError(t, err)
	as := listener.Addr().String()
	require.True(t, strings.HasPrefix(as, "127.0.0.1:"))
	require.Equal(t, "127.0.0.1", host)
	require.Equal(t, as, fmt.Sprintf("%s:%s", host, port))
}
