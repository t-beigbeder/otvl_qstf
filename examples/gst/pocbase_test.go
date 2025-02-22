package gst

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSimpleRoundTrip(t *testing.T) {
	require.NoError(t, SimpleRoundTrip())
}

func TestLargeRoundTrip(t *testing.T) {
	require.NoError(t, LargeRoundTrip())
}

func TestTwoReadersRoundTrip(t *testing.T) {
	require.NoError(t, TwoReadersRoundTrip())
}

func TestSimuFuncRoundTrip(t *testing.T) {
	require.NoError(t, SimuFuncRoundTrip())
}
