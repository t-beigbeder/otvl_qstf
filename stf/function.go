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
	GetInStreams() []InStream
	GetOutStreams() []OutStream
	Run() error
	Start() error
	Wait() error
	Terminate()
	Close()
	State() FunctionState
	Options() FcOptions
	Error() error
}

type StartWaiter interface {
	Start(context.Context) error
	Wait(context.Context) error
}

type function struct {
	id         string
	ctx        context.Context
	sw         StartWaiter
	mux        sync.Mutex
	state      FunctionState
	terminable bool
	ctrChan    chan struct{}
	err        error
	ins        map[string]*inStream
	outs       map[string]*outStream
	inList     []*inStream
	outList    []*outStream
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

func (fc *function) allocStId(opts *StOptions) {
	if opts.Id != "" {
		return
	}
	for i := 0; ; i++ {
		name := fmt.Sprintf("#%d", i)
		_, iOk := fc.ins[name]
		_, oOk := fc.outs[name]
		if !iOk && !oOk {
			opts.Id = name
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

func (fc *function) GetInStreams() []InStream {
	res := make([]InStream, 0, len(fc.inList))
	for _, in := range fc.inList {
		res = append(res, in)
	}
	return res
}

func (fc *function) GetOutStreams() []OutStream {
	res := make([]OutStream, 0, len(fc.outList))
	for _, out := range fc.outList {
		res = append(res, out)
	}
	return res
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
	fc.allocStId(&sopt.StOptions)
	is := &inStream{
		stream: stream{name: sopt.Id, ctx: fc.ctx},
		rr:     rr,
		opts:   sopt,
	}
	fc.ins[sopt.StOptions.Id] = is
	fc.inList = append(fc.inList, is)
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
	fc.allocStId(&sopt.StOptions)
	os := &outStream{
		stream: stream{name: sopt.Id, ctx: fc.ctx},
		wr:     wr,
		opts:   sopt,
	}
	fc.outs[sopt.StOptions.Id] = os
	fc.outList = append(fc.outList, os)
	fc.activateStream(&os.stream, os)
	return os, nil
}

func (fc *function) Start() error {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if err := fc.mustBeInState(StateInit); err != nil {
		return fc.setState(StateFinished, err)
	}
	if err := fc.sw.Start(fc.ctx); err != nil {
		return fc.setState(StateFinished, err)
	}
	fc.setState(StateStarted, nil)
	if !fc.terminable {
		return nil
	}

	fc.ctrChan = make(chan struct{}, 1)
	fc.wg.Add(1)
	go func() {
		defer fc.wg.Done()
		for {
			select {
			case <-fc.ctrChan:
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
	if err := fc.sw.Wait(fc.ctx); err != nil {
		return fc.setState(StateFinished, err)
	}
	fc.wg.Wait()
	var err error
	for _, in := range fc.inList {
		if in.err != nil {
			err = errors.Join(err, in.err)
		}
	}
	for _, out := range fc.outList {
		if out.err != nil {
			err = errors.Join(err, out.err)
		}
	}
	if err != nil {
		return fc.setState(StateFinished, err)
	}
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
		select {
		case fc.ctrChan <- struct{}{}:
		default:
			return
		}
	}
}

func (fc *function) Close() {
	for _, in := range fc.ins {
		in.stream.Terminate()
	}
	for _, out := range fc.outs {
		out.stream.Terminate()
	}
	fc.Terminate()
}

func (fc *function) State() FunctionState {
	return fc.state
}

func (fc *function) Options() FcOptions {
	return FcOptions{Id: fc.id, Terminable: fc.terminable}
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
		id:         fopt.Id,
		terminable: fopt.Terminable,
		ctx:        ctx,
		sw:         sw,
		ins:        make(map[string]*inStream, 1),
		outs:       make(map[string]*outStream, 1),
	}
	return fc, nil
}
