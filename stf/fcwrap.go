package stf

import (
	"context"
	"encoding/json"
	"io"
)

type FuncWrapper struct {
	function
	ain  any
	aout any
}

var _ Function = &FuncWrapper{}

func NewFuncWrapper(ctx context.Context, wrapped func(any) any, in io.ReadCloser, out io.WriteCloser) *FuncWrapper {
	fc := &FuncWrapper{function: function{ctx: ctx}}
	fc.sw = NewDefaultStartWaiter(&fc.function)
	fc.AddInStreamUnmarshaller(in,
		json.Unmarshal,
		func(a any) error {
			fc.ain = a
			return nil
		})
	fc.inStreams[0].setOnce = true
	fc.AddOutStreamMarshaller(out,
		func() (any, error) {
			return wrapped(fc.aout), nil
		},
		json.Marshal,
	)
	fc.outStreams[0].getOnce = true
	return fc
}
