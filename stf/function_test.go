package stf

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"testing"
	"time"
)

type testSw struct {
	is      InStream
	results chan string
}

func (sw *testSw) Start(_ context.Context) error {
	sw.is.Start()
	return nil
}

func (t testSw) Wait(_ context.Context) error {
	return nil
}

var _ StartWaiter = &testSw{}

func TestTerminable(t *testing.T) {
	sw := &testSw{results: make(chan string, 1)}
	ctx := context.Background()
	fc, err := NewFunction(
		ctx,
		sw,
		FcTerminable(true),
	)
	require.NoError(t, err)

	out := bfio.NewBufWr()
	oss, err := fc.AddOutStream(out, OstDiscrete(true),
		OstBGet(
			func(_ context.Context) ([]byte, error) {
				result, ok := <-sw.results
				if !ok {
					return nil, errors.New("OstBGet no more data")
				}
				return []byte(result), nil
			}))
	require.NoError(t, err)
	oss.Start()

	ttbs := []byte{}
	for i := 0; i < 10; i++ {
		msg := []byte(fmt.Sprintf("msg %d", i))
		lbs := make([]byte, 4)
		binary.BigEndian.PutUint32(lbs, uint32(len(msg)))
		ttbs = append(ttbs, lbs...)
		ttbs = append(ttbs, msg...)
	}
	in := bytes.NewReader(ttbs)

	is, err := fc.AddInStream(in, IstDiscrete(true),
		IstBSet(
			func(_ context.Context, bs []byte) error {
				sw.results <- string(bs)
				time.Sleep(5 * time.Millisecond)
				return nil
			}))
	require.NoError(t, err)
	go func() {
		time.Sleep(100 * time.Millisecond)
		fc.Terminate()
		close(sw.results)
	}()
	sw.is = is
	err = fc.Run()
	require.Contains(t, err.Error(), "EOF")
	require.Contains(t, err.Error(), "OstBGet no more data")
}
