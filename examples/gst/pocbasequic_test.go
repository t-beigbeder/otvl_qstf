package gst

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewQHost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h1, h2, err := setupQHosts(ctx, nil, nil)
	require.NoError(t, err)
	_, _ = h1, h2
	cancel()
	time.Sleep(time.Millisecond * 100)
}

func TestSimpleQuicRoundTrip(t *testing.T) {
	err := SimpleQuicRoundTrip()
	require.NoError(t, err)
}

func TestLargeQuicRoundTrip(t *testing.T) {
	err := LargeQuicRoundTrip()
	require.NoError(t, err)
}

func TestTwoReadersQuicRoundTrip(t *testing.T) {
	err := TwoReadersQuicRoundTrip()
	require.NoError(t, err)
}

func TestSimuFuncQuicRoundTrip(t *testing.T) {
	err := SimuFuncQuicRoundTrip()
	require.NoError(t, err)
}

func TestSimuFuncQuicLargeRoundTrip(t *testing.T) {
	err := SimuFuncQuicLargeRoundTrip()
	require.NoError(t, err)
}
