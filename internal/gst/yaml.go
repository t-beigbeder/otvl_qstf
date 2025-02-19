package gst

import (
	"github.com/reugn/go-streams"
	"github.com/reugn/go-streams/flow"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"log"
)

type YamlSource[T any] struct {
	fileName string
	in       chan any
	sample   func() []T
}

var _ streams.Source = (*YamlSource[any])(nil)

func NewYamlSource[T any](fileName string, sample func() []T) *YamlSource[T] {
	source := &YamlSource[T]{
		fileName: fileName,
		in:       make(chan any),
		sample:   sample,
	}
	source.init()
	return source
}

func (ys *YamlSource[T]) init() {
	go func() {
		data := ys.sample()
		err := common.YamlLoad(ys.fileName, &data)
		if err != nil {
			log.Fatal(err)
			return
		}
		for _, v := range data {
			ys.in <- v
		}
		close(ys.in)
	}()
}

func (ys *YamlSource[T]) Via(operator streams.Flow) streams.Flow {
	flow.DoStream(ys, operator)
	return operator
}

func (ys *YamlSource[T]) Out() <-chan any {
	return ys.in
}
