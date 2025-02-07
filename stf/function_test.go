package stf

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type testSw struct {
	is      InStream
	results chan string
}

func (sw *testSw) Start() error {
	sw.is.Start()
	return nil
}

func (t testSw) Wait() error {
	return nil
}

var _ StartWaiter = &testSw{}

func TestTerminable(t *testing.T) {
	ctx := context.Background()
	sw := &testSw{results: make(chan string, 1)}
	fc, err := NewFunction(
		sw,
		FcTerminable(true),
	)
	require.NoError(t, err)

	out := newBufWr()
	oss, err := fc.AddOutStream(ctx, out, OstDiscrete(true),
		OstBGet(
			func() ([]byte, error) {
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

	is, err := fc.AddInStream(ctx, in, IstDiscrete(true),
		IstBSet(
			func(bs []byte) error {
				sw.results <- string(bs)
				time.Sleep(40 * time.Millisecond)
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
	require.NoError(t, err)
}
