package appctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/stf"
	os2 "os"
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

func TestGetFDesc(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		as.Catalog().DeclareFunction(
			FunctionDesc{Name: "TestGetFDesc"},
			&stf.WrappedFunction{
				json.Marshal, json.Unmarshal,
				func() any {
					a := ""
					return &a
				},
				func() any {
					a := ""
					return &a
				},
				func(ctx context.Context, a any) any {
					return fmt.Sprintf("TestGetFDesc: %v", a)
				},
			}, nil, nil)
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	fd, err := ac.GetFDesc("TestGetFDesc")
	require.NoError(t, err)
	require.NotNil(t, fd)
	require.Equal(t, "TestGetFDesc", fd.Name)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunSyncBase(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestNewAppClientRunSyncBase",
			func(ctx context.Context, a *string, err error) string {
				return fmt.Sprintf("TestNewAppClientRunSyncBase: %v", *a)
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 5; i++ {
		var sOut string
		_, err = ac.RunSyncFunction("TestNewAppClientRunSyncBase", "", MarshalJson, fmt.Sprintf("#%03d", i), &sOut)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("TestNewAppClientRunSyncBase: #%03d", i), sOut)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunSyncTyped(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[Tin, Tout]("TestNewAppClientRunSyncTyped",
			func(ctx context.Context, a *Tin, err error) Tout {
				return Tout{Result: fmt.Sprintf("TestNewAppClientRunSyncTyped: %v", a.Name)}
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 5; i++ {
		var sOut Tout
		_, err = ac.RunSyncFunction("TestNewAppClientRunSyncTyped", "", MarshalJson, Tin{fmt.Sprintf("#%03d", i)}, &sOut)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("TestNewAppClientRunSyncTyped: #%03d", i), sOut.Result)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunSyncReqLarge(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestNewAppClientRunSyncReqLarge",
			func(ctx context.Context, a *string, err error) string {
				return fmt.Sprintf("TestNewAppClientRunSyncReqLarge: %v", *a)
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	genLargeString := func() string {
		res := ""
		for i := 0; len(res) <= MaxInPlSize; i++ {
			s := fmt.Sprintf("#%03d ", i)
			res += s
		}
		return res
	}
	var sOut string
	in := genLargeString()
	_, err = ac.RunSyncFunction("TestNewAppClientRunSyncReqLarge", "", MarshalJson, in, &sOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "payload too large")
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunSyncRspLarge(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestNewAppClientRunSyncRspLarge",
			func(ctx context.Context, a *string, err error) string {
				return fmt.Sprintf("TestNewAppClientRunSyncRspLarge: %v", *a)
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	var sOut string
	genNotSoLargeString := func() string {
		res := ""
		for i := 0; ; i++ {
			s := fmt.Sprintf("TestNewAppClientRunSyncRspLarge #%03d ", i)
			if len(res)+len(s) >= MaxInPlSize {
				return res
			}
			res += s
		}
	}
	_, err = ac.RunSyncFunction("TestNewAppClientRunSyncRspLarge", "", MarshalJson, genNotSoLargeString(), &sOut)
	require.Error(t, err)
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientRunSyncSlow(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestNewAppClientRunSyncSlow",
			func(ctx context.Context, a *string, err error) string {
				time.Sleep(20 * time.Millisecond)
				return fmt.Sprintf("TestNewAppClientRunSyncSlow: %v", *a)
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	var sOut string
	_, err = ac.RunSyncFunction("TestNewAppClientRunSyncSlow", "", MarshalJson, "#000", &sOut)
	require.NoError(t, err)
	require.Equal(t, "TestNewAppClientRunSyncSlow: #000", sOut)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

type testSw struct {
	in any
}

func (sw *testSw) Start(ctx context.Context) error {
	cn := CurrentConnection(ctx)
	cn.GetLogger().Debug("testSw Start", "cn", cn.GetId())
	fc := CurrentFunction(ctx)
	cn.GetLogger().Debug("testSw Start", "fc", fc.Options())
	fc.GetInStreams()[0].Start()
	return nil
}

func (sw *testSw) Wait(ctx context.Context) error {
	cn := CurrentConnection(ctx)
	cn.GetLogger().Debug("testSw Wait", "cn", cn.GetId())
	fc := CurrentFunction(ctx)
	cn.GetLogger().Debug("testSw Wait", "fc", fc.Options())
	return nil
}

func getTestSwSthsRaw() *StreamHandlers {
	return &StreamHandlers{
		ihs: []IStreamHandler{
			{
				BSet: func(ctx context.Context, bytes []byte) error {
					CurrentLogger(ctx).Debug("getTestSwSths bset", "bytes", bytes)
					vls := CurrentValues(ctx)
					if vls == nil {
						return errors.New("no values")
					}
					vls["a"] = string(bytes)
					oss := CurrentFunction(ctx).GetOutStreams()
					oss[len(oss)-1].Start()
					return nil
				},
			},
		},
		ohs: []OStreamHandler{
			{
				BGet: func(ctx context.Context) ([]byte, error) {
					CurrentLogger(ctx).Debug("getTestSwSths bget")
					vls := CurrentValues(ctx)
					if vls == nil {
						return nil, errors.New("no values")
					}
					a, ok := vls["a"]
					if !ok {
						return nil, errors.New("no values")
					}
					CurrentLogger(ctx).Debug("getTestSwSths bget", "a", a)
					return []byte(fmt.Sprintf("response to %s", a)), nil
				},
			},
		},
	}
}

func TestNewAppClientRunNTermFuncRaw(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := RawFuncDeclarer(t.Name(), false)(as.Catalog())
		if err != nil {
			t.Fatal(err)
		}
	})
	require.NoError(t, err)
	_, fc, is, os, err := NewAppClientWithFuncStdio(port, t.Name())

	bgRes := ""
	go func() {
		js, err := toJsonBytes("hello world TestNewAppClientRunNTermFuncRaw")
		if err != nil {
			fmt.Fprintf(os2.Stderr, "toJsonBytes: %s\n", err)
			return
		}
		_, err = os.Write(js)
		if err != nil {
			fmt.Fprintf(os2.Stderr, "os.Write: %s\n", err)
			return
		}
		time.Sleep(20 * time.Millisecond)

		bs, err := fromBytes(is)
		if err != nil {
			fmt.Fprintf(os2.Stderr, "fromBytes: %s\n", err)
			return
		}
		bgRes = string(bs)
		fmt.Fprintf(os2.Stderr, "fromBytes: %s\n", string(bs))

	}()

	err = fc.Run()
	require.NoError(t, err)
	time.Sleep(40 * time.Millisecond)
	require.Equal(t, "response to \"hello world TestNewAppClientRunNTermFuncRaw\"", bgRes)
	cancel()
}

func TestNewAppClientRunTermFunc(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := JsonFuncDeclarer[string, string](t.Name(), true)(as.Catalog())
		if err != nil {
			t.Fatal(err)
		}
	})
	require.NoError(t, err)
	_, fc, is, os, err := NewAppClientWithFuncStdio(port, t.Name())
	require.NoError(t, err)

	bgRes := ""
	go func() {
		js, err := toJsonBytes("hello world")
		if err != nil {
			fmt.Fprintf(os2.Stderr, "toJsonBytes: %s\n", err)
			return
		}
		_, err = os.Write(js)
		if err != nil {
			fmt.Fprintf(os2.Stderr, "os.Write: %s\n", err)
			return
		}
		time.Sleep(100 * time.Millisecond)

		res := ""
		err = fromJsonBytes(is, &res)
		if err != nil {
			fmt.Fprintf(os2.Stderr, "fromJsonBytes: %s\n", err)
			return
		}
		fmt.Fprintf(os2.Stderr, "fromJsonBytes: %s\n", res)
		bgRes = res

		fc.Terminate()
	}()

	time.Sleep(40 * time.Millisecond)
	err = fc.Run()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)
	require.Equal(t, "response to hello world", bgRes)

	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewAppClientSWTermFunc(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := as.Catalog().DeclareFunction(
			FunctionDesc{
				Name:       "TestNewAppClientSWTermFunc",
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
			GetTSWSthsJson[string, string](),
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
	fc, err := ac.NewFunction("TestNewAppClientSWTermFunc", "")
	require.NoError(t, err)
	err = fc.AddOStream(os)
	require.NoError(t, err)
	err = fc.AddIStream(is)
	require.NoError(t, err)

	go func() {
		js, err := toJsonBytes("hello world")
		if err != nil {
			fmt.Fprintf(os2.Stderr, "toJsonBytes: %s\n", err)
			return
		}
		_, err = os.Write(js)
		if err != nil {
			fmt.Fprintf(os2.Stderr, "os.Write: %s\n", err)
			return
		}
		time.Sleep(100 * time.Millisecond)

		res := ""
		err = fromJsonBytes(is, &res)
		if err != nil {
			fmt.Fprintf(os2.Stderr, "fromJsonBytes: %s\n", err)
			return
		}
		fmt.Fprintf(os2.Stderr, "fromJsonBytes: %s\n", res)

		fc.Terminate()
	}()

	err = fc.Start()
	require.NoError(t, err)
	err = fc.Wait()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)

	cancel()
	time.Sleep(20 * time.Millisecond)
}
