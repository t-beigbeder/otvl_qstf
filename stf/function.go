package stf

import (
	"context"
	"errors"
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
	AddInStream(io.Reader, ...IstOption) (InStream, error)
	GetInStream(string) InStream
	AddOutStream(io.Writer, ...OstOption) (OutStream, error)
	GetOutStream(string) OutStream
	Run() error
	Start() error
	Wait() error
	Terminate()
	State() FunctionState
	Error() error
}

type StartWaiter interface {
	Start() error
	Wait() error
}

type function struct {
	name       string
	ctx        context.Context
	sw         StartWaiter
	mux        sync.Mutex
	state      FunctionState
	terminable bool
	ctrChan    chan struct{}
	err        error
	ins        map[string]*inStream
	outs       map[string]*outStream
	wg         sync.WaitGroup
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

func (fc *function) allocStName(opts *StOptions) {
	if opts.Name != "" {
		return
	}
	for i := 0; ; i++ {
		name := fmt.Sprintf("#%d", i)
		_, iOk := fc.ins[name]
		_, oOk := fc.outs[name]
		if !iOk && !oOk {
			opts.Name = name
			break
		}
	}
}

func (fc *function) GetInStream(s string) InStream {
	is, _ := fc.ins[s]
	return is
}

func (fc *function) GetOutStream(s string) OutStream {
	os, _ := fc.outs[s]
	return os
}

func (fc *function) activateStream(st *stream, sti Stream) {
	fc.wg.Add(1)
	st.ctrChan = make(chan ctrlMsg, 1)
	go func() {
		defer fc.wg.Done()
		st.loop(sti)
	}()
}

func (fc *function) AddInStream(rr io.Reader, opts ...IstOption) (InStream, error) {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateInit); err != nil {
		return nil, fc.setState(StateFinished, err)
	}
	var (
		sopt IstOptions
		err  error
	)

	for _, opt := range opts {
		if iErr := opt(&sopt); iErr != nil {
			err = errors.Join(err, iErr)
		}
	}
	if err != nil {
		return nil, fc.setState(StateFinished, err)
	}
	fc.allocStName(&sopt.StOptions)
	is := &inStream{
		stream: stream{name: sopt.Name, ctx: fc.ctx},
		rr:     rr,
		opts:   sopt,
	}
	fc.ins[sopt.StOptions.Name] = is
	fc.activateStream(&is.stream, is)
	return is, nil
}

func (fc *function) AddOutStream(wr io.Writer, opts ...OstOption) (OutStream, error) {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateInit); err != nil {
		return nil, fc.setState(StateFinished, err)
	}
	var (
		sopt OstOptions
		err  error
	)

	for _, opt := range opts {
		if iErr := opt(&sopt); iErr != nil {
			err = errors.Join(err, iErr)
		}
	}
	if err != nil {
		return nil, fc.setState(StateFinished, err)
	}
	fc.allocStName(&sopt.StOptions)
	os := &outStream{
		stream: stream{name: sopt.Name, ctx: fc.ctx},
		wr:     wr,
		opts:   sopt,
	}
	fc.outs[sopt.StOptions.Name] = os
	fc.activateStream(&os.stream, os)
	return os, nil
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
	fc.setState(StateStarted, nil)
	if !fc.terminable {
		return nil
	}

	fc.wg.Add(1)
	fc.ctrChan = make(chan struct{}, 1)
	go func() {
		defer fc.wg.Done()
		for {
			select {
			case <-fc.ctrChan:
				for _, in := range fc.ins {
					in.stream.Terminate()
				}
				for _, out := range fc.outs {
					out.stream.Terminate()
				}
				fc.setState(StateFinished, nil)
				return
			}
		}
	}()
	return nil
}

func (fc *function) Wait() error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateStarted); err != nil {
		return fc.setState(StateFinished, err)
	}
	if err := fc.sw.Wait(); err != nil {
		return fc.setState(StateFinished, err)
	}
	fc.wg.Wait()
	return fc.setState(StateFinished, nil)
}

func (fc *function) Run() error {
	if err := fc.Start(); err != nil {
		return err
	}
	return fc.Wait()
}

func (fc *function) Terminate() {
	if fc.terminable {
		fc.ctrChan <- struct{}{}
	}
}

func (fc *function) State() FunctionState {
	return fc.state
}

func (fc *function) Error() error {
	return fc.err
}

func NewFunction(ctx context.Context, sw StartWaiter, opts ...FcOption) (Function, error) {
	var (
		fopt FcOptions
		err  error
	)
	for _, opt := range opts {
		if iErr := opt(&fopt); iErr != nil {
			err = errors.Join(err, iErr)
		}
	}
	if err != nil {
		return nil, err
	}
	fc := &function{
		name:       fopt.Name,
		terminable: fopt.Terminable,
		ctx:        ctx,
		sw:         sw,
		ins:        make(map[string]*inStream, 1),
		outs:       make(map[string]*outStream, 1),
	}
	return fc, nil
}
