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

func TestNewAppClientRunSync(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		as.Catalog().DeclareFunction(
			FunctionDesc{Name: "TestNewAppClientRunSync"},
			&stf.WrappedFunction{
				json.Marshal, json.Unmarshal,
				templateForString, templateForString,
				func(ctx context.Context, a any) any {
					return fmt.Sprintf("TestNewAppClientRunSync: %v", a)
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
		err = ac.RunSyncFunction("TestNewAppClientRunSync", "", fmt.Sprintf("#%03d", i), &sOut)
		require.NoError(t, err)
	}
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

func getTestSwSthsJson() *StreamHandlers {
	return &StreamHandlers{
		ihs: []IStreamHandler{
			{
				BSet: func(ctx context.Context, bytes []byte) error {
					CurrentLogger(ctx).Debug("getTestSwSths bset", "bytes", bytes)
					vls := CurrentValues(ctx)
					if vls == nil {
						return errors.New("no values")
					}
					return nil
				},
				Unmarshal: json.Unmarshal,
				NewASet: func() any {
					v := ""
					return &v
				},
				ASet: func(ctx context.Context, a any) error {
					CurrentLogger(ctx).Debug("getTestSwSths aset", "a", *(a.(*string)))
					vls := CurrentValues(ctx)
					if vls == nil {
						return errors.New("no values")
					}
					vls["a"] = *(a.(*string))
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
				AGet: func(ctx context.Context) (any, error) {
					CurrentLogger(ctx).Debug("getTestSwSths aget")
					vls := CurrentValues(ctx)
					if vls == nil {
						return nil, errors.New("no values")
					}
					a, ok := vls["a"]
					if !ok {
						return nil, errors.New("no values")
					}
					CurrentLogger(ctx).Debug("getTestSwSths aget", "a", a)
					return fmt.Sprintf("response to %s", a), nil
				},
				Marshaller: json.Marshal,
			},
		},
	}
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

func TestNewAppClientRunNTermFunc(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := as.Catalog().DeclareFunction(
			FunctionDesc{
				Name:       "TestNewAppClientRunNTermFunc",
				Terminable: false,
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
			getTestSwSthsRaw(),
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
	fc, err := ac.NewFunction("TestNewAppClientRunNTermFunc", "")
	require.NoError(t, err)
	err = fc.AddOStream(os)
	require.NoError(t, err)
	err = fc.AddIStream(is)
	require.NoError(t, err)

	go func() {
		js, err := toJsonBytes("hello world TestNewAppClientRunNTermFunc")
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

		bs, err := fromBytes(is)
		if err != nil {
			fmt.Fprintf(os2.Stderr, "fromBytes: %s\n", err)
			return
		}
		fmt.Fprintf(os2.Stderr, "fromBytes: %s\n", string(bs))

	}()

	err = fc.Run()
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	cancel()
}

func TestNewAppClientRunTermFunc(t *testing.T) {
	port, cancel, err := RunTestServer(func(as AppServer) {
		err := as.Catalog().DeclareFunction(
			FunctionDesc{
				Name:       "TestNewAppClientRunTermFunc",
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
			getTestSwSthsJson(),
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
	fc, err := ac.NewFunction("TestNewAppClientRunTermFunc", "")
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

	time.Sleep(100 * time.Millisecond)
	err = fc.Run()
	require.NoError(t, err)
	err = fc.Terminate()
	require.NoError(t, err)

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
			getTestSwSthsJson(),
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
