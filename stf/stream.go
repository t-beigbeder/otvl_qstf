package stf

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type StreamState int

const (
	StStopped StreamState = iota
	StRunning
	StTerminated
	StCanceled
)

func (s StreamState) String() string {
	switch s {
	case StStopped:
		return "stopped"
	case StRunning:
		return "running"
	case StTerminated:
		return "terminated"
	case StCanceled:
		return "canceled"
	}
	panic("invalid state")
}

type looperOut struct {
	maxProcessed bool
	eof          bool
	err          error
}

var loNil = looperOut{}

type Stream interface {
	GetName() string
	GetState() StreamState
	Start()
	Stop()
	Terminate()
	looper() looperOut
	close() error
}

type InStream interface {
	Stream
	io.Reader
	Options() IstOptions
}

type OutStream interface {
	Stream
	io.Writer
	Options() OstOptions
}

type ctrlMsg int

const (
	msgStart ctrlMsg = iota
	msgStop
	msgTerminate
)

func (cm ctrlMsg) String() string {
	switch cm {
	case msgStart:
		return "start"
	case msgStop:
		return "stop"
	case msgTerminate:
		return "terminate"
	}
	panic("invalid ctrl message")
}

type stream struct {
	name          string
	state         StreamState
	discreteCount int
	ctx           context.Context
	ctrChan       chan ctrlMsg
	err           error
}

func (st *stream) GetName() string {
	return st.name
}

func (st *stream) GetState() StreamState {
	return st.state
}

func (st *stream) loop(sti Stream) {
	defer func() {
		cerr := sti.close()
		if cerr != nil {
			st.err = errors.Join(st.err, cerr)
		}
	}()

	for {
		select {
		case <-st.ctx.Done():
			st.state = StCanceled
			return
		case msg := <-st.ctrChan:
			switch msg {
			case msgStart:
				st.state = StRunning
				lo := sti.looper()
				if lo != loNil {
					if lo.err != nil {
						st.err = errors.Join(st.err, lo.err)
					}
					st.state = StTerminated
					return
				}
			case msgStop:
				st.state = StStopped
			case msgTerminate:
				st.state = StTerminated
				return
			}
		}
	}
}

func (st *stream) Start() {
	st.ctrChan <- msgStart
}

func (st *stream) Stop() {
	st.ctrChan <- msgStop
}

func (st *stream) Terminate() {
	st.ctrChan <- msgTerminate
}

type inStream struct {
	stream
	rr   io.Reader
	opts IstOptions
	read int
}

var _ InStream = &inStream{}

func (is *inStream) Options() IstOptions {
	return is.opts
}

func (is *inStream) readRaw() looperOut {
	bSize := 128
	if is.opts.BSize > 0 {
		bSize = is.opts.BSize
	}
	bs := make([]byte, bSize)
	read, err := is.Read(bs)
	is.read += read
	if read != 0 && is.opts.BSet != nil {
		if iErr := is.opts.BSet(is.ctx, bs[:read]); iErr != nil {
			err = iErr
		}
	}
	if err != nil && err != io.EOF {
		return looperOut{err: err}
	}
	return looperOut{eof: err == io.EOF}
}

func (is *inStream) readDiscrete() looperOut {
	bs := make([]byte, 4)
	read, err := io.ReadFull(is, bs)
	is.read += read
	if err != nil {
		return looperOut{err: err}
	}
	bln := binary.BigEndian.Uint32(bs)
	if is.opts.MaxLen > 0 && int(bln) > is.opts.MaxLen {
		return looperOut{err: fmt.Errorf("max length exceeded: %d > %d", bln, is.opts.MaxLen)}
	}
	bs = make([]byte, bln)
	read, err = io.ReadFull(is, bs)
	is.read += read
	if read != 0 && is.opts.BSet != nil {
		if iErr := is.opts.BSet(is.ctx, bs[:read]); iErr != nil {
			return looperOut{err: iErr}
		}
	}
	if is.opts.Unmarshaller != nil {
		v := is.opts.NewASet()
		if iErr := is.opts.Unmarshaller(bs[:read], v); iErr != nil {
			return looperOut{err: iErr}
		}
		if iErr := is.opts.ASet(is.ctx, v); iErr != nil {
			return looperOut{err: iErr}
		}
	}
	is.discreteCount++
	return loNil
}

func (is *inStream) looper() looperOut {
	for {
		var lo looperOut
		if is.opts.Discrete {
			lo = is.readDiscrete()
		} else {
			lo = is.readRaw()
		}
		if lo.err != nil {
			return looperOut{err: lo.err}
		}
		if is.opts.MaxNb > 0 && is.discreteCount >= is.opts.MaxNb {
			return looperOut{maxProcessed: true}
		}
	}
}

func (is *inStream) close() error {
	return nil
}

func (is *inStream) Read(p []byte) (n int, err error) {
	select {
	case <-is.ctx.Done():
		return 0, is.ctx.Err()
	case msg := <-is.ctrChan:
		if msg == msgTerminate {
			is.state = StTerminated
			return 0, ErrUnexpectedTerminate
		}
		break
	default:
		break
	}
	return is.rr.Read(p)
}

type outStream struct {
	stream
	wr      io.Writer
	opts    OstOptions
	written int
}

var _ OutStream = &outStream{}

func (os *outStream) Options() OstOptions {
	return os.opts
}

func (os *outStream) writeRaw() looperOut {
	if os.opts.BGet == nil {
		return looperOut{err: fmt.Errorf("no BGet specified")}
	}
	bs, err := os.opts.BGet(os.ctx)
	if err != nil {
		return looperOut{err: err}
	}
	written := 0
	written, err = os.Write(bs)
	os.written += written
	return looperOut{err: err}
}

func (os *outStream) writeDiscrete() looperOut {
	var (
		bs  []byte
		v   any
		err error
	)
	if os.opts.AGet == nil {
		if os.opts.BGet == nil {
			return looperOut{err: fmt.Errorf("no BGet specified")}
		}
		if bs, err = os.opts.BGet(os.ctx); err != nil {
			return looperOut{err: err}
		}
	} else {
		if v, err = os.opts.AGet(os.ctx); err != nil {
			return looperOut{err: err}
		}
		if bs, err = os.opts.Marshaller(v); err != nil {
			return looperOut{err: err}
		}
	}
	if os.opts.MaxLen > 0 && len(bs) > os.opts.MaxLen {
		return looperOut{err: fmt.Errorf("max length exceeded: %d > %d", len(bs), os.opts.MaxLen)}
	}
	wbs := make([]byte, len(bs)+4)
	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
	copy(wbs[4:], bs)
	if _, err = os.Write(wbs); err != nil {
		return looperOut{err: err}
	}
	os.written += len(bs) + 4
	os.discreteCount++
	return loNil
}

func (os *outStream) looper() looperOut {
	for {
		var lo looperOut
		if os.opts.Discrete {
			lo = os.writeDiscrete()
		} else {
			lo = os.writeRaw()
		}
		if lo.err != nil {
			return looperOut{err: lo.err}
		}
		if os.opts.MaxNb > 0 && os.discreteCount >= os.opts.MaxNb {
			return looperOut{maxProcessed: true}
		}
	}
}

func (os *outStream) close() error {
	if !os.opts.CloseOnTerminate {
		return nil
	}
	cer, ok := os.wr.(io.Closer)
	if !ok {
		return fmt.Errorf("writer does not implement Closer (%T)", os.wr)
	}
	return cer.Close()
}

func (os *outStream) Write(p []byte) (n int, err error) {
	select {
	case <-os.ctx.Done():
		return 0, os.ctx.Err()
	case msg := <-os.ctrChan:
		if msg == msgTerminate {
			os.state = StTerminated
			return 0, ErrUnexpectedTerminate
		}
		break
	default:
		break
	}
	return os.wr.Write(p)
}
