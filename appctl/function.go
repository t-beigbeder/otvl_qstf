package appctl

import (
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"sync"
)

type MarshalerType int

const (
	MarshalerNone MarshalerType = iota
	MarshalerJSON
	MarshalerProtobuf
)

type StreamDesc struct {
	Name      string        `json:"name"`
	Discrete  bool          `json:"discrete"`
	MaxLen    int           `json:"maxlen"`
	MaxNb     int           `json:"maxnb"`
	Marshaler MarshalerType `json:"marshaler"`
}

type IStreamDesc struct {
	StreamDesc `json:"-"`
	BSize      int `json:"bsize"`
}

type OStreamDesc struct {
	StreamDesc `json:"-"`
}

type FunctionDesc struct {
	Name       string        `json:"name"`
	Terminable bool          `json:"terminable"`
	IStreams   []IStreamDesc `json:"iStreams"`
	OStreams   []OStreamDesc `json:"oStreams"`
}

type FunctionCatalog struct {
	mux        sync.RWMutex
	fds        map[string]FunctionDesc
	sws        map[string]stf.StartWaiter
	wrappedFns map[string]*stf.WrappedFunction
}

func NewFunctionCatalog() *FunctionCatalog {
	cat := &FunctionCatalog{
		fds:        make(map[string]FunctionDesc),
		sws:        make(map[string]stf.StartWaiter),
		wrappedFns: make(map[string]*stf.WrappedFunction),
	}
	_ = DeclareStfsNewFunction(cat)
	return cat
}

func (cat *FunctionCatalog) DeclareFunction(
	fnDesc FunctionDesc,
	sw stf.StartWaiter,
	wrappedFn *stf.WrappedFunction,
) error {
	cat.mux.Lock()
	defer cat.mux.Unlock()
	fn := fnDesc.Name
	_, ok := cat.fds[fn]
	if ok {
		return fmt.Errorf("function %s already exists", fn)
	}
	cat.fds[fn] = fnDesc
	cat.sws[fn] = sw
	cat.wrappedFns[fn] = wrappedFn
	return nil
}

func (cat *FunctionCatalog) GetFunction(fName string) (*FunctionDesc, stf.StartWaiter, *stf.WrappedFunction, error) {
	fd, ok := cat.fds[fName]
	if !ok {
		return nil, nil, nil, fmt.Errorf("function %s not found", fName)
	}
	return &fd, cat.sws[fName], cat.wrappedFns[fName], nil
}
