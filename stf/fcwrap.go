package stf

import (
	"context"
	"encoding/json"
	"errors"
	"io"
)

type sfwStartWait struct {
	is            InStream
	wrappedCalled bool
	wrappedResult any
}

var _ StartWaiter = &sfwStartWait{}

func (sw *sfwStartWait) Start(_ context.Context) error {
	sw.is.Start()
	return nil
}

func (sw *sfwStartWait) Wait(_ context.Context) error {
	return nil
}

type WrappedFunction struct {
	Marshaller     func(any) ([]byte, error)
	Unmarshal      func(data []byte, v any) error
	InputTemplate  func() any
	OutputTemplate func() any
	Wrapped        func(context.Context, any) any
}

func NewSyncFuncWrapper(ctx context.Context, wf WrappedFunction, rr io.Reader, wr io.Writer, opts ...FcOption) (Function, error) {
	sw := &sfwStartWait{}
	fc, err := NewFunction(ctx, sw, opts...)
	if err != nil {
		return nil, err
	}
	if fc.Options().Terminable {
		return nil, errors.New("a sync function cannot be set terminable")
	}
	os, err := fc.AddOutStream(wr, OstDiscrete(true), OstMaxNb(1),
		OstAGet(
			func(context.Context) (any, error) {
				if !sw.wrappedCalled {
					return nil, errors.New("consume function not yet not called")
				}
				return sw.wrappedResult, nil
			},
			json.Marshal))
	if err != nil {
		return nil, err
	}
	is, err := fc.AddInStream(rr, IstDiscrete(true), IstMaxNb(1),
		IstASet(json.Unmarshal,
			wf.InputTemplate,
			func(ctx context.Context, ia any) error {
				sw.wrappedCalled = true
				sw.wrappedResult = wf.Wrapped(ctx, ia)
				os.Start()
				return nil
			}),
	)
	if err != nil {
		return nil, err
	}
	sw.is = is
	return fc, nil
}
