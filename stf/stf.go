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
	AddInStream(io.ReadCloser) error
	AddOutStream(io.WriteCloser) error
	AddOutGetter(func() ([]byte, error)) error
	AddOutUnmarshaler(func(data []byte, v any) error) error
	Run() error
	Start() error
	Wait() error
	State() FunctionState
	Error() error
}

type startWaiter interface {
	start() error
	wait() error
}

type function struct {
	mux        sync.Mutex
	state      FunctionState
	sw         startWaiter
	err        error
	ctx        context.Context
	inStreams  []io.ReadCloser
	outStreams []io.WriteCloser
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

func (fc *function) AddInStream(rc io.ReadCloser) error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	fc.inStreams = append(fc.inStreams, rc)
	return nil
}

func (fc *function) AddOutStream(wc io.WriteCloser) error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	fc.outStreams = append(fc.outStreams, wc)
	return nil
}

func (fc *function) Start() error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateInit); err != nil {
		return fc.setState(StateFinished, err)
	}
	if err := fc.sw.start(); err != nil {
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
	err := fc.sw.wait()
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
