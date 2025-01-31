package stf

import (
	"context"
	"io"
)

type StreamState int

const (
	StStopped StreamState = iota
	StRunning
	StTerminated
)

func (s StreamState) String() string {
	switch s {
	case StStopped:
		return "stopped"
	case StRunning:
		return "running"
	case StTerminated:
		return "terminated"
	}
	panic("invalid state")
}

type Stream interface {
	GetName() string
	GetState() StreamState
	loop()
	Start()
	Stop()
	Terminate()
}

type InStream interface {
	Stream
}

type OutStream interface {
	Stream
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
	name    string
	state   StreamState
	ctx     context.Context
	ctrChan chan ctrlMsg
	err     error
}

func (st *stream) GetName() string {
	return st.name
}

func (st *stream) GetState() StreamState {
	return st.state
}

func (st *stream) loop() {
	for {
		select {
		case <-st.ctx.Done():
			return
		case msg := <-st.ctrChan:
			switch msg {
			case msgStart:
				st.state = StRunning
				// TODO: run loop
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

type outStream struct {
	stream
	wr      io.Writer
	opts    OstOptions
	written int
}

var _ OutStream = &outStream{}
