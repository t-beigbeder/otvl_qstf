package appctl

import (
	"context"
	"encoding/json"
	"errors"
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
			&stf.WrappedFunction{
				func() any {
					a := ""
					return &a
				},
				func(ctx context.Context, a any) any {
					return fmt.Sprintf("TestNewAppClientBasic: %v", a)
				},
			}, nil, nil)
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

func getTestSwSths() *StreamHandlers {
	return &StreamHandlers{
		ihs: []IStreamHandler{
			{
				BSet: func(ctx context.Context, bytes []byte) error {
					return errors.New("not implemented")
				},
				Unmarshal: json.Unmarshal,
				NewASet: func() any {
					v := ""
					return &v
				},
				ASet: func(ctx context.Context, a any) error {
					return errors.New("not implemented")
				},
			},
		},
		ohs: []OStreamHandler{
			{
				BGet: func(ctx context.Context) ([]byte, error) {
					return nil, errors.New("not implemented")
				},
				AGet: func(ctx context.Context) (any, error) {
					return nil, errors.New("not implemented")
				},
				Marshaller: json.Marshal,
			},
		},
	}
}

func TestNewAppClientRunFunc(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := as.Catalog().DeclareFunction(
			FunctionDesc{
				Name:       "TestNewAppClientRunFunc",
				Terminable: true,
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
			&testSw{},
			getTestSwSths(),
		)
		if err != nil {
			t.Fatal(err)
		}
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
	err = fc.AddOStream(os)
	require.NoError(t, err)
	err = fc.AddIStream(is)
	require.NoError(t, err)
	err = fc.Run()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)

	fc, err = ac.NewFunction("TestNewAppClientRunFunc", "")
	require.NoError(t, err)
	err = fc.AddOStream(os)
	require.NoError(t, err)
	err = fc.AddIStream(is)
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
