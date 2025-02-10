package appctl

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRidbs(t *testing.T) {
	require.Equal(t, uint64(1), NewRidBs(1).Get())
	require.Equal(t, uint64(0x123456789abcdef0), NewRidBs(0x123456789abcdef0).Get())
	require.Equal(t, uint64(0x9abcdef012345678), NewRidBs(0x9abcdef012345678).Get())
	require.Equal(t, "123456789abcdef0", NewRidBs(0x123456789abcdef0).String())
	require.Equal(t, "9abcdef012345678", NewRidBs(0x9abcdef012345678).String())
	rds := NewRidBs(0x123456789abcdef0)
	SetRidBs(uint64(0x9abcdef012345678), rds[:])
	require.Equal(t, uint64(0x9abcdef012345678), rds.Get())
	SetLenBs(uint32(0x12345678), rds[0:4])
	SetLenBs(uint32(0x9abcdef0), rds[4:8])
	require.Equal(t, uint64(0x123456789abcdef0), rds.Get())
	SetLenBs(uint32(0x9abcdef0), rds[0:])
	SetLenBs(uint32(0x12345678), rds[4:])
	require.Equal(t, uint64(0x9abcdef012345678), rds.Get())
	require.Equal(t, uint32(0x9abcdef0), LenBs(rds[0:]).Get())
	require.Equal(t, uint32(0x12345678), LenBs(rds[4:]).Get())
}
