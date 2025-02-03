package netutils

import (
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestGetIPPort(t *testing.T) {
	ip, p, err := GetIPPort(":0")
	require.NoError(t, err)
	var zip net.IP
	require.Equal(t, zip, ip)
	require.Equal(t, 0, p)
}
