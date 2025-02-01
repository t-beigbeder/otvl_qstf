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

func (sw *sfwStartWait) Start() error {
	sw.is.Start()
	return nil
}

func (sw *sfwStartWait) Wait() error {
	return nil
}

func NewSyncFuncWrapper(ctx context.Context, wrapped func(any) any, rr io.Reader, wr io.Writer, opts ...FcOption) (Function, error) {
	sw := &sfwStartWait{}
	fc, err := NewFunction(ctx, sw, opts...)
	if err != nil {
		return nil, err
	}
	os, err := fc.AddOutStream(wr, OstDiscrete(true), OstMaxNb(1),
		OstAGet(
			func() (any, error) {
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
		IstASet(json.Unmarshal, func(ia any) error {
			sw.wrappedCalled = true
			sw.wrappedResult = wrapped(ia)
			os.Start()
			return nil
		}))
	if err != nil {
		return nil, err
	}
	sw.is = is
	return fc, nil
}
