package appctl

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"io"
	"log/slog"
	"os"
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

func templateForString() any {
	a := ""
	return &a
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
	fc.GetInStreams()[0].Start()
	return nil
}

func (sw *TSW) Wait(ctx context.Context) error {
	cn := CurrentConnection(ctx)
	cn.GetLogger().Debug("TSW Wait", "cn", cn.GetId())
	fc := CurrentFunction(ctx)
	cn.GetLogger().Debug("TSW Wait", "fc", fc.Options())
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
			},
			nil, &TSW{}, GetTSWSthsRaw())
	}
}

func JsonFuncDeclarer[TI any, TO any](fName string, terminable bool) func(*FunctionCatalog) error {
	return func(cat *FunctionCatalog) error {
		return cat.DeclareFunction(
			FunctionDesc{
				Name:       fName,
				Terminable: terminable,
				IStreams: []IStreamDesc{
					{
						StreamDesc: StreamDesc{
							Name:     "in",
							Discrete: true,
							MaxNb:    1,
						},
					},
				},
				OStreams: []OStreamDesc{
					{
						StreamDesc: StreamDesc{
							Name:     "out",
							Discrete: true,
							MaxNb:    1,
						},
					},
				},
			},
			nil,
			&TSW{}, GetTSWSthsRaw())
	}
}
