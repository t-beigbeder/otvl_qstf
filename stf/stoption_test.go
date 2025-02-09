package stf

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStId(t *testing.T) {
	nist := func(id string, opts ...IstOption) IstOptions {
		var os IstOptions
		for _, o := range opts {
			o(&os)
		}
		return os
	}
	os := nist("id", IstId("a"), IstBsize(256))
	require.Equal(t, IstOptions{StOptions: StOptions{Id: "a"}, BSize: 256}, os)
}
