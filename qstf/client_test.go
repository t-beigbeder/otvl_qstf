package qstf

import (
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"log/slog"
	"testing"
	"time"
)

func TestClientBasic(t *testing.T) {
	td := t.TempDir()
	cancel, err := RunQstfTestClientServer(td,
		func(sh *ServerHost) error {
			return sh.RegisterFunction("theFunc", func(r *RStream) {
				logger := common.GetLoggerFor("theApp")
				ist, err := common.LStReader(r)
				if err != nil {
					logger.Error("theFunc error LStReader", "err", err)
					return
				}
				logger.Info("theFunc read", "ist", ist)
			})
		},
		func(ch *ClientHost, cnt Connector, logger *slog.Logger) error {
			st, err := ch.OpenStream(cnt, "", "theFunc")
			if err != nil {
				return err
			}
			err = common.LstWriter(st, "written TestClientBasic")
			if err != nil {
				return err
			}
			time.Sleep(10 * time.Millisecond)
			err = st.Close()
			require.NoError(t, err)
			return nil
		},
	)
	require.NoError(t, err)
	cancel()
}

func TestSimpleUseCase(t *testing.T) {
	// StrSrc(args) -> Flow(client) -> QstSnk(func, args) -> (net)
	// (net) -> QstSrc(func,args) -> Flow(func) -> StdoutSnk

}

func TestBasicSideChannel(t *testing.T) {
	// Same as TestSimpleUseCase but args are passed on a side channel
}
