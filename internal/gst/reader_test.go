package gst

import (
	"bytes"
	ext "github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/examples/gst"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"testing"
)

func TestLbsRW(t *testing.T) {
	ds := [][]byte{
		{},
		[]byte("Hello TestLbsRW!"),
	}
	for _, dt := range ds {
		rr := bytes.NewReader(common.Bs2LBs(dt))
		bs, err := LBsReader(rr)
		require.NoError(t, err)
		require.Equal(t, dt, bs)
	}
}

func TestNewReaderSource(t *testing.T) {
	rr := bytes.NewReader(common.Bs2LBs([]byte("Hello TestNewReaderSource!")))
	source, err := NewReaderSource[[]byte](rr, LBsReader)
	if err != nil {
		return
	}
	source.Via(gst.AsStringFlow()).To(ext.NewStdoutSink())
}
