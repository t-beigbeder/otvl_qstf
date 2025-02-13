package appctl

import (
	"context"
	"encoding/binary"
	"encoding/json"
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
