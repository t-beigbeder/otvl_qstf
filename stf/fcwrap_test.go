package stf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"testing"
	"time"
)

func TestNewSyncFuncWrapperBasic(t *testing.T) {
	wbs, err := toJsonBytes("value for test")
	require.NoError(t, err)
	in := bytes.NewReader(wbs)
	out := bfio.NewBufWr()
	fcw, err := NewSyncFuncWrapper(context.Background(),
		WrappedFunction{
			json.Marshal, json.Unmarshal, templateForString, templateForString,
			func(_ context.Context, a any) any {
				return "response for " + *(a.(*string))
			},
		},
		in, out)
	require.NoError(t, err)
	err = fcw.Run()
	require.NoError(t, err)
	var a string
	err = fromJsonBytes(out, &a)
	require.Equal(t, "response for value for test", a)
}

func TestNewSyncFuncWrapperSlow(t *testing.T) {
	wbs, err := toJsonBytes("value for test")
	require.NoError(t, err)
	in := bytes.NewReader(wbs)
	out := bfio.NewBufWr()
	fcw, err := NewSyncFuncWrapper(context.Background(),
		WrappedFunction{
			json.Marshal, json.Unmarshal, templateForString, templateForString,
			func(_ context.Context, a any) any {
				time.Sleep(time.Millisecond * 200)
				return "response for " + *(a.(*string))
			},
		},
		in, out)
	require.NoError(t, err)
	err = fcw.Run()
	require.NoError(t, err)
	var a string
	err = fromJsonBytes(out, &a)
	require.Equal(t, "response for value for test", a)
}

func TestNewSyncFuncWrapperLarge(t *testing.T) {
	type dst struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var din []dst
	for i := 0; i < 100; i++ {
		din = append(din, dst{Key: fmt.Sprintf("k%03d", i), Value: fmt.Sprintf("v%03d", i)})
	}
	wbs, err := toJsonBytes(din)
	require.NoError(t, err)
	in := bytes.NewReader(wbs)
	out := bfio.NewBufWr()
	fcw, err := NewSyncFuncWrapper(context.Background(),
		WrappedFunction{
			json.Marshal, json.Unmarshal,
			func() any { return &[]dst{} },
			func() any { return &[]dst{} },
			func(_ context.Context, a any) any { return a },
		},
		in, out)
	require.NoError(t, err)
	err = fcw.Run()
	require.NoError(t, err)
	var a []dst
	err = fromJsonBytes(out, &a)
	require.Equal(t, din, a)
}
