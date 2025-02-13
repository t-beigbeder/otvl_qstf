package appctl

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"sync"
)

type StreamDesc struct {
	Name     string `json:"name"`
	Discrete bool   `json:"discrete"`
	MaxLen   int    `json:"maxlen"`
	MaxNb    int    `json:"maxnb"`
}

type IStreamDesc struct {
	StreamDesc `json:""`
	BSize      int `json:"bsize"`
}

type OStreamDesc struct {
	StreamDesc `json:""`
}

type Marshaller int

const (
	MarshalNone Marshaller = iota
	MarshalJson
	MarshalYaml
)

type WrapperDesc struct {
	InMarshaller  Marshaller `json:"inMarshaller"`
	OutMarshaller Marshaller `json:"outMarshaller"`
}

type FunctionDesc struct {
	Name       string        `json:"name"`
	Terminable bool          `json:"terminable"`
	IStreams   []IStreamDesc `json:"iStreams"`
	OStreams   []OStreamDesc `json:"oStreams"`
	Wrapper    WrapperDesc   `json:"wrapper"`
}

type IStreamHandler struct {
	BSet      func(context.Context, []byte) error
	Unmarshal func(data []byte, v any) error
	NewASet   func() any
	ASet      func(context.Context, any) error
}

type OStreamHandler struct {
	BGet       func(context.Context) ([]byte, error)
	AGet       func(context.Context) (any, error)
	Marshaller func(any) ([]byte, error)
}

type StreamHandlers struct {
	ihs []IStreamHandler
	ohs []OStreamHandler
}

type FunctionCatalog struct {
	mux        sync.Mutex
	fds        map[string]FunctionDesc
	wrappedFns map[string]*stf.WrappedFunction
	sws        map[string]stf.StartWaiter
	sths       map[string]*StreamHandlers
}

func NewFunctionCatalog() *FunctionCatalog {
	cat := &FunctionCatalog{
		fds:        make(map[string]FunctionDesc),
		wrappedFns: make(map[string]*stf.WrappedFunction),
		sws:        make(map[string]stf.StartWaiter),
		sths:       make(map[string]*StreamHandlers),
	}
	return cat
}

func (cat *FunctionCatalog) DeclareFunction(
	fnDesc FunctionDesc,
	wrappedFn *stf.WrappedFunction,
	sw stf.StartWaiter,
	sths *StreamHandlers,
) error {
	cat.mux.Lock()
	defer cat.mux.Unlock()
	fn := fnDesc.Name
	_, ok := cat.fds[fn]
	if ok {
		return fmt.Errorf("function %s already exists", fn)
	}
	if sths != nil && len(sths.ihs) != len(fnDesc.IStreams) {
		return fmt.Errorf("function %s describes %d iStreams, but declares %d handlers",
			fn, len(fnDesc.IStreams), len(sths.ihs))
	}
	if sths != nil && len(sths.ohs) != len(fnDesc.OStreams) {
		return fmt.Errorf("function %s describes %d oStreams, but declares %d handlers",
			fn, len(fnDesc.OStreams), len(sths.ohs))
	}
	if sths == nil && len(fnDesc.IStreams) > 0 {
		return fmt.Errorf("function %s describes %d iStreams, but does not declare handlers",
			fn, len(fnDesc.IStreams))
	}
	if sths == nil && len(fnDesc.OStreams) > 0 {
		return fmt.Errorf("function %s describes %d oStreams, but does not declare handlers",
			fn, len(fnDesc.OStreams))
	}
	cat.fds[fn] = fnDesc
	cat.wrappedFns[fn] = wrappedFn
	cat.sws[fn] = sw
	cat.sths[fn] = sths
	return nil
}

func (cat *FunctionCatalog) GetFunction(fName string) (*FunctionDesc, stf.StartWaiter, *stf.WrappedFunction, *StreamHandlers, error) {
	fd, ok := cat.fds[fName]
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("function %s not found", fName)
	}
	return &fd, cat.sws[fName], cat.wrappedFns[fName], cat.sths[fName], nil
}

func factoryFor[T any]() any {
	var a T
	return &a
}

func JsonSyncFuncDeclarer[TI any, TO any](fName string, wrp func(context.Context, *TI, error) TO) func(*FunctionCatalog) error {
	return func(cat *FunctionCatalog) error {
		return cat.DeclareFunction(
			FunctionDesc{
				Name:    fName,
				Wrapper: WrapperDesc{InMarshaller: MarshalJson, OutMarshaller: MarshalJson},
			},
			&stf.WrappedFunction{
				json.Marshal, json.Unmarshal,
				factoryFor[TI], factoryFor[TO],
				func(ctx context.Context, a any) any {
					var err error
					ta, ok := a.(*TI)
					if !ok {
						err = fmt.Errorf("%T is not %T", a, *new(TI))
					}
					return wrp(ctx, ta, err)
				},
			},
			nil, nil)
	}
}
