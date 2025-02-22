package gst

import (
	"errors"
	"fmt"
	"github.com/reugn/go-streams"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"io"
	"math"
)

type ElementWriter[T any] func(io.Writer, T) error

type WriterSink[T any] struct {
	writer   io.WriteCloser
	elWriter ElementWriter[T]
	in       chan any
	done     chan struct{}
}

var _ streams.Sink = (*WriterSink[any])(nil)

func NewWriterSink[T any](writer io.WriteCloser, elWriter ElementWriter[T]) (*WriterSink[T], error) {
	if writer == nil {
		return nil, errors.New("nil writer")
	}
	ws := &WriterSink[T]{
		writer:   writer,
		elWriter: elWriter,
		in:       make(chan any),
		done:     make(chan struct{}),
	}
	ws.init()
	return ws, nil
}

func (ws *WriterSink[T]) init() {
	// FIXME: write, close and type conversion errors to be notified
	go func() {
		defer ws.writer.Close()
		defer close(ws.done)
		for el := range ws.in {
			err := ws.elWriter(ws.writer, el.(T))
			if err != nil {
				return
			}
		}
	}()
}

func (ws *WriterSink[T]) In() chan<- any {
	return ws.in
}

func (ws *WriterSink[T]) AwaitCompletion() {
	<-ws.done
}

func PTWriter(wr io.Writer, bs []byte) error {
	_, err := wr.Write(bs)
	return err
}

func LBsWriter(wr io.Writer, ebs []byte) error {
	if len(ebs) > math.MaxUint16 {
		return fmt.Errorf("buffer too big for uint16 %d", len(ebs))
	}
	_, err := wr.Write(common.Bs2LBs(ebs))
	return err
}

func LStWriter(wr io.Writer, st string) error {
	return LBsWriter(wr, []byte(st))
}
