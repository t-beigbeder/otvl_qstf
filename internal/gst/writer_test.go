package gst

import (
	"bytes"
	"github.com/reugn/go-streams/flow"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"testing"
)

func TestLBsWR(t *testing.T) {
	ds := [][]byte{
		{},
		[]byte("Hello TestLBsWR!"),
	}
	for _, dt := range ds {
		bf := bfio.NewBufWr()
		err := LBsWriter(bf, dt)
		require.NoError(t, err)
		bs, err := LBsReader(bytes.NewReader(bf.Bytes()))
		require.NoError(t, err)
		require.Equal(t, dt, bs)
	}
}

func TestNewWriterSink(t *testing.T) {
	msgsBs := common.LBsSample(t.Name(), true)
	rr := bytes.NewReader(msgsBs)
	source, err := NewReaderSource[[]byte](rr, LBsReader)
	if err != nil {
		return
	}

	wr := bfio.NewBufWr()
	sink, err := NewWriterSink[[]byte](wr, LBsWriter)
	if err != nil {
		return
	}
	source.Via(flow.NewPassThrough()).To(sink)
	require.Equal(t, msgsBs, wr.Bytes())
}
