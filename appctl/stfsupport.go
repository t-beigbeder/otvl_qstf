package appctl

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/stf"
)

func DeclareStfsNewFunction(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameNewFunction}, nil,
		&stf.WrappedFunction{
			func() any {
				return &NewFunctionReqMsg{}
			},
			func(ctx context.Context, in any) any {
				res := NewFunctionRespMsg{}
				req, ok := in.(*NewFunctionReqMsg)
				if !ok {
					res.Error = fmt.Sprintf("NewFunction: invalid input type %T", in)
					return res
				}
				acn := ctx.Value("cn")
				cn, ok := (acn).(Connection)
				if acn == nil || !ok {
					res.Error = fmt.Sprintf("Function %s cannot be run as connection is unknown", req.Name)
					return res
				}
				fd, sw, wf, err := cat.GetFunction(req.Name)
				if err != nil {
					res.Error = err.Error()
					return res
				}
				if wf != nil {
					res.Error = fmt.Sprintf("Function %s is wrapped and must be run with RunSyncFunction", fd.Name)
					return res
				}
				if sw == nil {
					res.Error = fmt.Sprintf("Function %s cannot be run as it doesn't have StartWaiter interface defined", fd.Name)
					return res
				}
				fc, err := stf.NewFunction(ctx, sw)
				if err != nil {
					res.Error = err.Error()
					return res
				}
				err = cn.NewFunction(req.Id, fc, fd)
				res.Desc = *fd
				return res
			},
		},
	)
}

func DeclareStfsFuncAddIStream(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameFuncAddIStream}, nil,
		&stf.WrappedFunction{
			func() any {
				return &FuncStreamReqMsg{}
			},
			func(ctx context.Context, in any) any {
				res := &FuncStreamRespMsg{}
				req, ok := in.(*FuncStreamReqMsg)
				if !ok {
					res.Error = fmt.Sprintf("%s: invalid input type %T", FNameFuncAddIStream, in)
					return res
				}
				acn := ctx.Value("cn")
				cn, ok := (acn).(Connection)
				if acn == nil || !ok {
					res.Error = fmt.Sprintf("Function %s stream %s cannot be added as connection is unknown", req.FcId, req.StId)
					return res
				}
				fc, fd := cn.GetFunction(req.FcId)
				if fc == nil || fd == nil {
					res.Error = fmt.Sprintf("Function %s does not exist", req.FcId)
					return res
				}
				is := cn.GetIStream(req.StId)
				if is == nil {
					res.Error = fmt.Sprintf("Stream in %s does not exist", req.StId)
					return res
				}
				if len(fc.GetInStreams()) >= len(fd.IStreams) {
					res.Error = fmt.Sprintf("Stream in %s has no descriptor", req.StId)
				}
				opts := []stf.IstOption{}
				opts = append(opts, stf.IstDiscrete(true))
				opts = append(opts,
					stf.IstASet(json.Unmarshal,
						func() any {
							return nil
						},
						func(ctx context.Context, a any) error {
							return nil
						}),
				)

				_, err := fc.AddInStream(is, opts...)
				if err != nil {
					res.Error = err.Error()
					return res
				}
				return res
			},
		},
	)
}

func DeclareStfsFuncAddOStream(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameFuncAddOStream}, nil,
		&stf.WrappedFunction{
			func() any {
				return &FuncStreamReqMsg{}
			},
			func(ctx context.Context, in any) any {
				res := &FuncStreamRespMsg{}
				req, ok := in.(*FuncStreamReqMsg)
				if !ok {
					res.Error = fmt.Sprintf("%s: invalid input type %T", FNameFuncAddIStream, in)
					return res
				}
				acn := ctx.Value("cn")
				cn, ok := (acn).(Connection)
				if acn == nil || !ok {
					res.Error = fmt.Sprintf("Function %s stream %s cannot be added as connection is unknown", req.FcId, req.StId)
					return res
				}
				fc, fd := cn.GetFunction(req.FcId)
				if fc == nil || fd == nil {
					res.Error = fmt.Sprintf("Function %s does not exist", req.FcId)
					return res
				}
				os := cn.GetOStream(req.StId)
				if os == nil {
					res.Error = fmt.Sprintf("Stream out %s does not exist", req.StId)
					return res
				}
				if len(fc.GetOutStreams()) >= len(fd.OStreams) {
					res.Error = fmt.Sprintf("Stream out %s has no descriptor", req.StId)
				}
				opts := []stf.OstOption{}
				opts = append(opts, stf.OstDiscrete(true))
				opts = append(opts,
					stf.OstAGet(
						func(ctx context.Context) (any, error) {
							return nil, nil
						},
						json.Marshal,
					))
				_, err := fc.AddOutStream(os, opts...)
				if err != nil {
					res.Error = err.Error()
					return res
				}
				return res
			},
		},
	)
}

func getWrappedFuncOperate(fName string, verb string, doer func(stf.Function) error) *stf.WrappedFunction {
	return &stf.WrappedFunction{
		func() any {
			return &FuncOperateReqMsg{}
		},
		func(ctx context.Context, in any) any {
			res := &FuncOperateRespMsg{}
			req, ok := in.(*FuncOperateReqMsg)
			if !ok {
				res.Error = fmt.Sprintf("%s: invalid input type %T", fName, in)
				return res
			}
			acn := ctx.Value("cn")
			cn, ok := (acn).(Connection)
			if acn == nil || !ok {
				res.Error = fmt.Sprintf("Function %s cannot be %s as connection is unknown", req.Id, verb)
				return res
			}
			fc, fd := cn.GetFunction(req.Id)
			if fc == nil || fd == nil {
				res.Error = fmt.Sprintf("Function %s does not exist", req.Id)
				return res
			}
			if len(fc.GetInStreams()) != len(fd.IStreams) {
				res.Error = fmt.Sprintf("instreams should be %d, got %d", len(fd.IStreams), len(fc.GetInStreams()))
				return res
			}
			if len(fc.GetOutStreams()) != len(fd.OStreams) {
				res.Error = fmt.Sprintf("outstreams should be %d, got %d", len(fd.IStreams), len(fc.GetInStreams()))
				return res
			}
			err := doer(fc)
			if err != nil {
				res.Error = err.Error()
				return res
			}
			return res
		},
	}
}

func DeclareStfsFuncRun(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameFuncRun}, nil,
		getWrappedFuncOperate("FuncRun", "run", func(fc stf.Function) error { return fc.Run() }),
	)
}

func DeclareStfsFuncStart(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameFuncStart}, nil,
		getWrappedFuncOperate("FuncStart", "started", func(fc stf.Function) error { return fc.Start() }),
	)
}
func DeclareStfsFuncWait(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameFuncWait}, nil,
		getWrappedFuncOperate("FuncWait", "awaited", func(fc stf.Function) error { return fc.Wait() }),
	)
}

func DeclareStfsFuncTerminate(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameFuncTerminate}, nil,
		getWrappedFuncOperate("FuncTerminate", "terminated", func(fc stf.Function) error { fc.Terminate(); return nil }),
	)
}
