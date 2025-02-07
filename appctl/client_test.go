package appctl

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"testing"
	"time"
)

func TestNewAppClientBasic(t *testing.T) {
	port, cancel, err := RunTestServer(nil)
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunSync(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		as.Catalog().DeclareFunction(
			FunctionDesc{Name: "TestNewAppClientBasic"},
			nil,
			&stf.WrappedFunction{
				func() any {
					a := ""
					return &a
				},
				func(ctx context.Context, a any) any {
					return fmt.Sprintf("TestNewAppClientBasic: %v", a)
				},
			})
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 5; i++ {
		var sOut string
		err = ac.RunSyncFunction("TestNewAppClientBasic", fmt.Sprintf("#%03d", i), &sOut)
		require.NoError(t, err)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
}

type testSw struct {
	in any
}

func (sw *testSw) Start(ctx context.Context) error {
	acn := ctx.Value("cn")
	cn := acn.(Connection)
	if cn != nil {
		cn.GetLogger().Debug("testSw Start")
	}
	return nil
}

func (sw *testSw) Wait(ctx context.Context) error {
	acn := ctx.Value("cn")
	cn := acn.(Connection)
	if cn != nil {
		cn.GetLogger().Debug("testSw Wait")
	}
	return nil
}

func TestNewAppClientRunFunc(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		as.Catalog().DeclareFunction(
			FunctionDesc{Name: "TestNewAppClientRunFunc"},
			&testSw{},
			nil,
		)
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	os, err := ac.AddIStream("")
	require.NoError(t, err)
	is, err := ac.GetOStream("")
	require.NoError(t, err)
	_, _ = is, os
	fc, err := ac.NewFunction("TestNewAppClientRunFunc", "")
	require.NoError(t, err)
	err = fc.Run()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)

	fc, err = ac.NewFunction("TestNewAppClientRunFunc", "")
	require.NoError(t, err)
	err = fc.Start()
	require.NoError(t, err)
	err = fc.Wait()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)

	cancel()
	time.Sleep(20 * time.Millisecond)
}
