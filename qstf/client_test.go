package qstf

import (
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/qstf/quicutils"
	"testing"
	"time"
)

func TestClientBasic(t *testing.T) {
	td := t.TempDir()
	sh, port, cancel, err := RunQstfTestServerWithCtc(td, nil)
	require.NoError(t, err)
	require.NotNil(t, sh)
	require.NotNil(t, port)
	time.Sleep(time.Millisecond * 10)
	ch := NewClientHost(common.GetLoggerFor("TestClientBasic"), "TestClientBasicHostId")
	require.NotNil(t, ch)
	cnt, err := ch.AddConnector(":"+port, sh.HostId, &quicutils.QuicOptions{}, 0)
	require.NoError(t, err)
	require.NotNil(t, cnt)
	cancel()
}

func TestSimpleUseCase(t *testing.T) {
	// StrSrc(args) -> Flow(client) -> QstSnk(func, args) -> (net)
	// (net) -> QstSrc(func,args) -> Flow(func) -> StdoutSnk

}

func TestBasicSideChannel(t *testing.T) {
	// Same as TestSimpleUseCase but args are passed on a side channel
}
