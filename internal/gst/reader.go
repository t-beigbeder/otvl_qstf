package gst

import (
	"errors"
	"fmt"
	"github.com/reugn/go-streams"
	"github.com/reugn/go-streams/flow"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"io"
	"math"
)

type ElementReader[T any] func(io.Reader) (T, error)

type ReaderSource[T any] struct {
	reader   io.Reader
	elReader ElementReader[T]
	out      chan any
}

var _ streams.Source = (*ReaderSource[any])(nil)

func NewReaderSource[T any](reader io.Reader, elReader ElementReader[T]) (*ReaderSource[T], error) {
	if reader == nil {
		return nil, errors.New("nil reader")
	}
	source := &ReaderSource[T]{
		reader:   reader,
		elReader: elReader,
		out:      make(chan any),
	}
	source.init()
	return source, nil
}

func (rs *ReaderSource[T]) init() {
	go func() {
		defer close(rs.out)
		for {
			e, err := rs.elReader(rs.reader)
			if err != nil {
				return
			}
			rs.out <- e
		}
	}()
}

func (rs *ReaderSource[T]) Out() <-chan any {
	return rs.out
}

func (rs *ReaderSource[T]) Via(operator streams.Flow) streams.Flow {
	flow.DoStream(rs, operator)
	return operator
}

func LBsReader(rr io.Reader) ([]byte, error) {
	lbs := common.I16Bs{}
	_, err := io.ReadFull(rr, lbs[:])
	if err != nil {
		return nil, err
	}
	ebs := make([]byte, lbs.Get())
	_, err = io.ReadFull(rr, ebs)
	if err != nil {
		return nil, err
	}
	return ebs, nil
}

func LBsWriter(wr io.Writer, ebs []byte) error {
	if len(ebs) > math.MaxUint16 {
		return fmt.Errorf("buffer too big for uint16 %d", len(ebs))
	}
	_, err := wr.Write(common.Bs2LBs(ebs))
	return err
}

func LStReader(rr io.Reader) (string, error) {
	ebs, err := LBsReader(rr)
	if err != nil {
		return "", err
	}
	return string(ebs), nil
}

func LStWriter(wr io.Writer, st string) error {
	return LBsWriter(wr, []byte(st))
}
