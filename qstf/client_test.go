package qstf

import (
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/t-beigbeder/otvl_qstf/qstf/quicutils"
	"testing"
	"time"
)

func TestClientBasic(t *testing.T) {
	td := t.TempDir()
	sh, port, cancel, cfs, err := RunQstfTestServerWithCtc(td, nil)
	require.NoError(t, err)
	require.NotNil(t, sh)
	require.NotNil(t, port)
	time.Sleep(time.Millisecond * 10)
	logger := common.GetLoggerFor("TestClientBasic")
	ch := NewClientHost(logger, "TestClientBasicHostId")
	require.NotNil(t, ch)
	qo := &quicutils.QuicOptions{
		TlsOptions: quicutils.TlsOptions{CACertFile: cfs["cac"]},
		Alpns:      netutils.NextProtosFor(QstfAlpn),
	}
	require.NoError(t, err)
	cnt, err := NewConnector("localhost:"+port, sh.HostId, qo, 0, logger)
	require.NoError(t, err)
	require.NotNil(t, cnt)
	st, err := ch.OpenStream(cnt, "", "theFunc")
	require.NoError(t, err)
	require.NotNil(t, st)
	err = st.Close()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 1000)
	cancel()
}

func TestSimpleUseCase(t *testing.T) {
	// StrSrc(args) -> Flow(client) -> QstSnk(func, args) -> (net)
	// (net) -> QstSrc(func,args) -> Flow(func) -> StdoutSnk

}

func TestBasicSideChannel(t *testing.T) {
	// Same as TestSimpleUseCase but args are passed on a side channel
}
