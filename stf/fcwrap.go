package stf

import (
	"context"
	"encoding/json"
	"io"
)

type fswt struct {
	wrapped func(any) any
}

func (f fswt) Start() error {
	//TODO implement me
	panic("implement me")
}

func (f fswt) Wait() error {
	//TODO implement me
	panic("implement me")
}

var fsw StartWaiter = &fswt{}

func NewFuncWrapper(ctx context.Context, wrapped func(any) any, rr io.Reader, wr io.Writer, opts ...FcOption) (Function, error) {
	sw := &fswt{wrapped: wrapped}
	fc, err := NewFunction(ctx, sw, opts...)
	if err != nil {
		return nil, err
	}
	fc.AddInStream(rr, IstMaxNb(1), IstASet(json.Unmarshal, func(a any) error {
		return nil
	}))
	return fc, nil
}
