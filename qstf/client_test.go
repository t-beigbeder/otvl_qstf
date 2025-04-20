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
		func(sh ServerHost) error {
			if err := sh.RegisterFunction("theFunc1", func(r *rStream) {
				logger := common.GetLoggerFor("theApp")
				ist, err := common.LStReader(r)
				if err != nil {
					logger.Error("theFunc1 error LStReader", "err", err)
					return
				}
				logger.Info("theFunc1 read", "ist", ist)
			}); err != nil {
				return err
			}
			if err := sh.RegisterFunction("theFunc2", func(r *rStream) {
				logger := common.GetLoggerFor("theApp")
				ist, err := common.LStReader(r)
				if err != nil {
					logger.Error("theFunc2 error LStReader", "err", err)
					return
				}
				logger.Info("theFunc2 read", "ist", ist)
			}); err != nil {
				return err
			}
			return nil
		},
		func(ch ClientHost, cnt Connector, logger *slog.Logger) error {
			st1, err := ch.OpenStream(cnt, "", "theFunc1")
			if err != nil {
				return err
			}
			err = common.LstWriter(st1, "written TestClientBasic")
			if err != nil {
				return err
			}
			st2, err := ch.OpenStream(cnt, "", "theFunc2")
			if err != nil {
				return err
			}
			err = common.LstWriter(st2, "written TestClientBasic")
			if err != nil {
				return err
			}
			err = st1.Close()
			if err != nil {
				return err
			}
			require.NoError(t, err)
			time.Sleep(100 * time.Millisecond) // FIXME
			err = st2.Close()
			if err != nil {
				return err
			}
			require.NoError(t, err)
			return nil
		},
	)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond) // FIXME
	cancel()
}

func TestSimpleUseCase(t *testing.T) {
	td := t.TempDir()
	cancel, err := RunQstfTestClientServer(td,
		func(sh ServerHost) error {
			if err := sh.RegisterFunction("theFuncThatLog", func(r *rStream) {
				logger := common.GetLoggerFor("theApp")
				ist, err := common.LStReader(r)
				if err != nil {
					logger.Error("theFuncThatLog error LStReader", "err", err)
					return
				}
				logger.Info("theFuncThatLog read", "ist", ist)
			}); err != nil {
				return err
			}
			return nil
		},
		func(ch ClientHost, cnt Connector, logger *slog.Logger) error {
			st, err := ch.OpenStream(cnt, "", "theFuncThatLog")
			if err != nil {
				return err
			}
			err = common.LstWriter(st, "written TestClientBasic")
			if err != nil {
				return err
			}
			err = st.Close()
			if err != nil {
				return err
			}
			require.NoError(t, err)
			return nil
		},
	)
	require.NoError(t, err)
	cancel()

}

func TestBasicSideChannel(t *testing.T) {
	// Same as TestSimpleUseCase but args are passed on a side channel
}
