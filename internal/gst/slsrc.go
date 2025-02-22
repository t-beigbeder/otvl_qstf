package gst

import (
	"github.com/reugn/go-streams"
	"github.com/reugn/go-streams/flow"
)

type SliceSource[T any] struct {
	content []T
	index   int
	out     chan any
}

var _ streams.Source = (*SliceSource[any])(nil)

func NewSliceSource[T any](content []T) *SliceSource[T] {
	s := &SliceSource[T]{
		content: content,
		out:     make(chan any),
	}
	s.init()
	return s
}

func (s *SliceSource[T]) init() {
	go func() {
		defer close(s.out)
		for {
			if s.index >= len(s.content) {
				return
			}
			s.out <- s.content[s.index]
			s.index++
		}
	}()
}

func (s *SliceSource[T]) Out() <-chan any {
	return s.out
}

func (s *SliceSource[T]) Via(ope streams.Flow) streams.Flow {
	flow.DoStream(s, ope)
	return ope
}
