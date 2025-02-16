package appctl

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"io"
	"log/slog"
	"os"
	"time"
)

type TASInizer func(AppServer)

func RunTestServer(initializer TASInizer) (string, context.CancelFunc, error) {
	var as AppServer
	port, cancel, err := netutils.RunTestServer(
		QstfAlpn,
		func(ctx context.Context, qc quic.Connection, logger *slog.Logger) {
			if as == nil {
				as = NewAppServer(ctx, NewFunctionCatalog(), logger)
				if initializer != nil {
					initializer(as)
				}
			}
			as.NewCnc(qc)
		}, GetLoggerFor("server"))
	return port, cancel, err
}

func GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func GetLoggerFor(app string) *slog.Logger {
	return GetLogger().With("app", app)
}

func toJsonBytes(a any) ([]byte, error) {
	bs, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	wbs := make([]byte, len(bs)+4)
	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
	copy(wbs[4:], bs)
	return wbs, err
}

func toBytes(bs []byte) []byte {
	wbs := make([]byte, len(bs)+4)
	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
	copy(wbs[4:], bs)
	return wbs
}

func fromBytes(rr io.Reader) ([]byte, error) {
	bs := make([]byte, 4)
	_, err := io.ReadFull(rr, bs)
	if err != nil {
		return nil, err
	}
	bln := binary.BigEndian.Uint32(bs)
	bs = make([]byte, bln)
	_, err = io.ReadFull(rr, bs)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func fromJsonBytes(rr io.Reader, a any) error {
	bs, err := fromBytes(rr)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, a)
}

type Tin struct {
	Name string `json:"name"`
}

type Tout struct {
	Result string `json:"result"`
}

type TSW struct{}

func (sw *TSW) Start(ctx context.Context) error {
	cn := CurrentConnection(ctx)
	cn.GetLogger().Debug("TSW Start", "cn", cn.GetId())
	fc := CurrentFunction(ctx)
	cn.GetLogger().Debug("TSW Start", "fc", fc.Options())
	cn.GetLogger().Debug("TSW Start", "fc", CurrentValues(ctx))
	fc.GetInStreams()[0].Start()
	return nil
}

func (sw *TSW) Wait(ctx context.Context) error {
	cn := CurrentConnection(ctx)
	cn.GetLogger().Debug("TSW Wait", "cn", cn.GetId())
	fc := CurrentFunction(ctx)
	cn.GetLogger().Debug("TSW Wait", "fc", fc.Options())
	ainV, ok := CurrentValues(ctx)["in-pl"]
	if ok {
		inV, ok := ainV.(*Tin)
		if ok {
			CurrentValues(ctx)["out-pl"] = &Tout{Result: "result for " + inV.Name}
		}
	}
	return nil
}

func GetTSWSthsRaw() *StreamHandlers {
	return &StreamHandlers{
		ihs: []IStreamHandler{
			{
				BSet: func(ctx context.Context, bytes []byte) error {
					CurrentLogger(ctx).Debug("GetTSWSthsRaw BSet", "bytes", len(bytes))
					vls := CurrentValues(ctx)
					if vls == nil {
						return errors.New("no values")
					}
					vls["bytes"] = bytes
					oss := CurrentFunction(ctx).GetOutStreams()
					oss[len(oss)-1].Start()
					return nil
				},
			},
		},
		ohs: []OStreamHandler{
			{
				BGet: func(ctx context.Context) ([]byte, error) {
					CurrentLogger(ctx).Debug("GetTSWSthsRaw BGet")
					vls := CurrentValues(ctx)
					if vls == nil {
						return nil, errors.New("no values")
					}
					bytes, ok := vls["bytes"].([]byte)
					if !ok {
						return nil, errors.New("no bytes in values")
					}
					CurrentLogger(ctx).Debug("GetTSWSthsRaw BGet", "bytes", len(bytes))
					rs := make([]byte, len(bytes)+len("response to "))
					copy(rs[0:], "response to ")
					copy(rs[len("response to "):], bytes)
					return rs, nil
				},
			},
		},
	}
}

func GetTSWSthsJson[TI any, TO any]() *StreamHandlers {
	return &StreamHandlers{
		ihs: []IStreamHandler{
			{
				Unmarshal: json.Unmarshal,
				NewASet:   func() any { return FactoryFor[TI]() },
				ASet: func(ctx context.Context, a any) error {
					CurrentLogger(ctx).Debug("GetTSWSthsJson ASet", "In", *(a.(*TI)))
					vls := CurrentValues(ctx)
					if vls == nil {
						return errors.New("no values")
					}
					vls["In"] = *(a.(*TI))
					oss := CurrentFunction(ctx).GetOutStreams()
					oss[len(oss)-1].Start()
					return nil
				},
			},
		},
		ohs: []OStreamHandler{
			{
				AGet: func(ctx context.Context) (any, error) {
					CurrentLogger(ctx).Debug("GetTSWSthsJson AGet")
					vls := CurrentValues(ctx)
					if vls == nil {
						return nil, errors.New("no values")
					}
					a, ok := vls["In"]
					if !ok {
						return nil, errors.New("no In in values")
					}
					CurrentLogger(ctx).Debug("GetTSWSthsJson AGet", "In", a)
					return fmt.Sprintf("response to %s", a), nil
				},
				Marshaller: json.Marshal,
			},
		},
	}
}

func getStdinDesc() []IStreamDesc {
	return []IStreamDesc{
		{
			StreamDesc: StreamDesc{
				Name:     "in",
				Discrete: true,
				MaxNb:    1,
			},
		},
	}
}

func getStdoutDesc() []OStreamDesc {
	return []OStreamDesc{
		{
			StreamDesc: StreamDesc{
				Name:     "out",
				Discrete: true,
				MaxNb:    1,
			},
		},
	}
}

func RawFuncDeclarer(fName string, terminable bool) func(*FunctionCatalog) error {
	return func(cat *FunctionCatalog) error {
		return cat.DeclareFunction(
			FunctionDesc{
				Name:       fName,
				Terminable: terminable,
				IStreams:   getStdinDesc(),
				OStreams:   getStdoutDesc(),
				Wrapper:    WrapperDesc{InMarshaller: MarshalJson, OutMarshaller: MarshalJson},
			},
			&stf.WrappedFunction{
				InputTemplate:  FactoryFor[Tin],
				OutputTemplate: FactoryFor[Tout],
			}, &TSW{}, GetTSWSthsRaw())
	}
}

func JsonFuncDeclarer[TI any, TO any](fName string, terminable bool) func(*FunctionCatalog) error {
	return func(cat *FunctionCatalog) error {
		return cat.DeclareFunction(
			FunctionDesc{
				Name:       fName,
				Terminable: terminable,
				IStreams:   getStdinDesc(),
				OStreams:   getStdoutDesc(),
			},
			nil,
			&TSW{}, GetTSWSthsJson[string, string]())
	}
}

func NewAppClientWithFuncStdio(port string, fName string) (AppClient, FcClient, IStream, OStream, error) {
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	os, err := ac.AddIStream("in")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	is, err := ac.GetOStream("out")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	fc, err := ac.NewFunction(fName, "")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = fc.AddOStream(os)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = fc.AddIStream(is)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return ac, fc, is, os, nil
}

func BgStdInOut(rr io.Reader, wr io.Writer, label string, isJson bool, out *string) {
	var err error
	bs := []byte("hello world " + label)
	if isJson {
		bs, err = toJsonBytes(string(bs))
		if err != nil {
			fmt.Fprintf(os.Stderr, "toJsonBytes: %s\n", err)
			return
		}
	} else {
		bs = toBytes(bs)
	}
	_, err = wr.Write(bs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "os.Write: %s\n", err)
		return
	}
	time.Sleep(20 * time.Millisecond)

	if isJson {
		err = fromJsonBytes(rr, out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fromBytes: %s\n", err)
			return
		}
	} else {
		bs, err = fromBytes(rr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fromBytes: %s\n", err)
			return
		}
		*out = string(bs)
	}
}
