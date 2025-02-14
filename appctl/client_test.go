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

func TestBasic(t *testing.T) {
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

func TestRunSyncBase(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestRunSyncBase",
			func(ctx context.Context, a *string, err error) string {
				return fmt.Sprintf("TestRunSyncBase: %v", *a)
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 5; i++ {
		var sOut string
		_, err = ac.RunSyncFunction("TestRunSyncBase", "", MarshalJson, fmt.Sprintf("#%03d", i), &sOut)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("TestRunSyncBase: #%03d", i), sOut)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestRunSyncTyped(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[Tin, Tout]("TestRunSyncTyped",
			func(ctx context.Context, a *Tin, err error) Tout {
				return Tout{Result: fmt.Sprintf("TestRunSyncTyped: %v", a.Name)}
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 5; i++ {
		var sOut Tout
		_, err = ac.RunSyncFunction("TestRunSyncTyped", "", MarshalJson, Tin{fmt.Sprintf("#%03d", i)}, &sOut)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("TestRunSyncTyped: #%03d", i), sOut.Result)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestRunSyncReqLarge(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestRunSyncReqLarge",
			func(ctx context.Context, a *string, err error) string {
				return fmt.Sprintf("TestRunSyncReqLarge: %v", *a)
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
	_, err = ac.RunSyncFunction("TestRunSyncReqLarge", "", MarshalJson, in, &sOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "payload too large")
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestRunSyncRspLarge(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestRunSyncRspLarge",
			func(ctx context.Context, a *string, err error) string {
				return fmt.Sprintf("TestRunSyncRspLarge: %v", *a)
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
			s := fmt.Sprintf("TestRunSyncRspLarge #%03d ", i)
			if len(res)+len(s) >= MaxInPlSize {
				return res
			}
			res += s
		}
	}
	_, err = ac.RunSyncFunction("TestRunSyncRspLarge", "", MarshalJson, genNotSoLargeString(), &sOut)
	require.Error(t, err)
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestRunSyncSlow(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		_ = JsonSyncFuncDeclarer[string, string]("TestRunSyncSlow",
			func(ctx context.Context, a *string, err error) string {
				time.Sleep(20 * time.Millisecond)
				return fmt.Sprintf("TestRunSyncSlow: %v", *a)
			})(as.Catalog())
	})
	require.NoError(t, err)
	ac, err := NewAppClient(context.Background(), "localhost:"+port, GetLoggerFor("client"))
	require.NoError(t, err)
	require.NotNil(t, ac)
	time.Sleep(10 * time.Millisecond)
	var sOut string
	_, err = ac.RunSyncFunction("TestRunSyncSlow", "", MarshalJson, "#000", &sOut)
	require.NoError(t, err)
	require.Equal(t, "TestRunSyncSlow: #000", sOut)
	cancel()
	time.Sleep(20 * time.Millisecond)
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

func TestRunNTermFuncRaw(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := RawFuncDeclarer(t.Name(), false)(as.Catalog())
		if err != nil {
			t.Fatal(err)
		}
	})
	require.NoError(t, err)
	_, fc, is, os, err := NewAppClientWithFuncStdio(port, t.Name())

	bgRes := ""
	go BgStdInOut(is, os, t.Name(), false, &bgRes)

	err = fc.Run()
	require.NoError(t, err)
	time.Sleep(40 * time.Millisecond)
	require.Equal(t, "response to hello world TestRunNTermFuncRaw", bgRes)
	cancel()
}

func TestRunTermFunc(t *testing.T) {
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
		BgStdInOut(is, os, t.Name(), true, &bgRes)
		fc.Terminate()
	}()

	time.Sleep(40 * time.Millisecond)
	err = fc.Run()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)
	require.Equal(t, "response to hello world "+t.Name(), bgRes)

	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestSWTermFunc(t *testing.T) {
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
		BgStdInOut(is, os, t.Name(), true, &bgRes)
		fc.Terminate()
	}()

	err = fc.Start()
	require.NoError(t, err)
	err = fc.Wait()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)
	require.Equal(t, "response to hello world "+t.Name(), bgRes)
	err = fc.Close()
	require.NoError(t, err)

	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestSWCloseFunc(t *testing.T) {
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
		BgStdInOut(is, os, t.Name(), true, &bgRes)
		fc.Close()
	}()

	err = fc.Start()
	require.NoError(t, err)
	err = fc.Wait()
	require.NoError(t, err)
	require.Equal(t, "response to hello world "+t.Name(), bgRes)

	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestSWCloseClient(t *testing.T) {
	port, _, err := RunTestServer(func(as AppServer) {
		err := JsonFuncDeclarer[string, string](t.Name(), true)(as.Catalog())
		if err != nil {
			t.Fatal(err)
		}
	})
	require.NoError(t, err)
	ac, fc, is, os, err := NewAppClientWithFuncStdio(port, t.Name())
	require.NoError(t, err)

	bgRes := ""
	go func() {
		BgStdInOut(is, os, t.Name(), true, &bgRes)
		ac.Close()
		time.Sleep(20 * time.Millisecond)
	}()

	err = fc.Start()
	require.NoError(t, err)
	err = fc.Wait()
	require.Error(t, err)
	require.Contains(t, err.Error(), "context canceled")
	time.Sleep(20 * time.Millisecond)
}

func TestCloseStream(t *testing.T) {
	port, _, err := RunTestServer(func(as AppServer) {
		err := JsonFuncDeclarer[string, string](t.Name(), true)(as.Catalog())
		if err != nil {
			t.Fatal(err)
		}
	})
	require.NoError(t, err)
	ac, fc, is, os, err := NewAppClientWithFuncStdio(port, t.Name())
	require.NoError(t, err)

	bgRes := ""
	go func() {
		BgStdInOut(is, os, t.Name(), true, &bgRes)
	}()

	err = fc.Start()
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)
	err = ac.CloseOStream(is.Id())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
	err = ac.CloseIStream(os.Id())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
	err = ac.CloseIStream(is.Id())
	require.Error(t, err)
	require.Contains(t, err.Error(), "owned by function")
	err = ac.CloseOStream(os.Id())
	require.Error(t, err)
	require.Contains(t, err.Error(), "owned by function")
	err = fc.Close()
	require.NoError(t, err)
	err = ac.CloseIStream(is.Id())
	require.NoError(t, err)
	err = ac.CloseOStream(os.Id())
	require.NoError(t, err)
}

func TestSWFuncRawPayload(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := RawFuncDeclarer(t.Name(), false)(as.Catalog())
		if err != nil {
			t.Fatal(err)
		}
	})
	require.NoError(t, err)
	_, fc, is, os, err := NewAppClientWithFuncStdio(port, t.Name())

	bgRes := ""
	go BgStdInOut(is, os, t.Name(), false, &bgRes)

	err = fc.StartWith(MarshalJson, &Tin{"TestSWFuncRawPayload"})
	require.NoError(t, err)
	vout := Tout{}
	err = fc.WaitWith(MarshalJson, &vout)
	require.NoError(t, err)
	require.Equal(t, "result for "+t.Name(), vout.Result)
	time.Sleep(40 * time.Millisecond)
	require.Equal(t, "response to hello world TestSWFuncRawPayload", bgRes)
	cancel()
}
