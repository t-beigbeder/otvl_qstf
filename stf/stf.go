package stf

import (
	"context"
	"fmt"
	"io"
	"sync"
)

type FunctionState int

const (
	StateInit FunctionState = iota
	StateStarted
	StateFinished
)

func (s FunctionState) String() string {
	switch s {
	case StateInit:
		return "initialized"
	case StateStarted:
		return "started"
	case StateFinished:
		return "finished"
	}
	panic("invalid state")
}

type Function interface {
	AddInStream(io.ReadCloser, func([]byte) error) error
	AddInStreamSetter(io.ReadCloser, func([]byte) error) error
	AddInStreamUnmarshaller(io.ReadCloser, func(data []byte, v any) error, func(any) error) error
	AddOutStream(io.WriteCloser, func() ([]byte, error)) error
	AddOutStreamGetter(io.WriteCloser, func() ([]byte, error)) error
	AddOutStreamMarshaller(io.WriteCloser, func() (any, error), func(any) ([]byte, error)) error
	Run() error
	Start() error
	Wait() error
	State() FunctionState
	Error() error
}

type StartWaiter interface {
	Start() error
	Wait() error
}

type istream struct {
	rc           io.ReadCloser
	isRaw        bool
	bset         func([]byte) error
	unmarshaller func(data []byte, v any) error
	aset         func(any) error
	read         int
	err          error
}

type ostream struct {
	wc         io.WriteCloser
	isRaw      bool
	bget       func() ([]byte, error)
	aget       func() (any, error)
	marshaller func(any) ([]byte, error)
	written    int
	err        error
}

type function struct {
	mux        sync.Mutex
	state      FunctionState
	sw         StartWaiter
	err        error
	ctx        context.Context
	inStreams  []istream
	outStreams []ostream
}

var _ Function = &function{}

func (fc *function) mustBeInState(state FunctionState) error {
	if fc.state != state {
		return fmt.Errorf("function state is not %s (%s)", state, fc.state)
	}
	return nil
}

func (fc *function) setState(newState FunctionState, err error) error {
	fc.state = newState
	fc.err = err
	return err
}

func (fc *function) addInStream(is istream) error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateInit); err != nil {
		return fc.setState(StateFinished, err)
	}
	fc.inStreams = append(fc.inStreams, is)
	return nil
}

func (fc *function) AddInStream(rc io.ReadCloser, set func([]byte) error) error {
	return fc.addInStream(istream{rc: rc, isRaw: true, bset: set})
}

func (fc *function) AddInStreamSetter(rc io.ReadCloser, set func([]byte) error) error {
	return fc.addInStream(istream{rc: rc, bset: set})
}

func (fc *function) AddInStreamUnmarshaller(rc io.ReadCloser, unmarshal func(data []byte, v any) error, set func(any) error) error {
	return fc.addInStream(istream{rc: rc, unmarshaller: unmarshal, aset: set})
}

func (fc *function) addOutStream(os ostream) error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateInit); err != nil {
		return fc.setState(StateFinished, err)
	}
	fc.outStreams = append(fc.outStreams, os)
	return nil
}

func (fc *function) AddOutStream(wc io.WriteCloser, get func() ([]byte, error)) error {
	return fc.addOutStream(ostream{wc: wc, isRaw: true, bget: get})
}

func (fc *function) AddOutStreamGetter(wc io.WriteCloser, get func() ([]byte, error)) error {
	return fc.addOutStream(ostream{wc: wc, bget: get})
}

func (fc *function) AddOutStreamMarshaller(wc io.WriteCloser, get func() (any, error), marshal func(any) ([]byte, error)) error {
	return fc.addOutStream(ostream{wc: wc, aget: get, marshaller: marshal})
}

func (fc *function) Start() error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateInit); err != nil {
		return fc.setState(StateFinished, err)
	}
	if err := fc.sw.Start(); err != nil {
		return fc.setState(StateFinished, err)
	}
	return fc.setState(StateStarted, nil)
}

func (fc *function) Wait() error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateStarted); err != nil {
		return fc.setState(StateFinished, err)
	}
	err := fc.sw.Wait()
	return fc.setState(StateFinished, err)
}

func (fc *function) Run() error {
	if err := fc.Start(); err != nil {
		return err
	}
	return fc.Wait()
}

func (fc *function) State() FunctionState {
	return fc.state
}

func (fc *function) Error() error {
	return fc.err
}
