package appctl

import (
	"context"
	"errors"
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
				err = cn.NewFunction(req.Id, fc)
				res.Desc = *fd
				return res
			},
		},
	)
}

func DeclareStfsFuncAddIStream(cat *FunctionCatalog) error {
	return errors.New("not yet implemented")
}

func DeclareStfsFuncAddOStream(cat *FunctionCatalog) error {
	return errors.New("not yet implemented")
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
			fc := cn.GetFunction(req.Id)
			if fc == nil {
				res.Error = fmt.Sprintf("Function %s does not exist", req.Id)
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
		FunctionDesc{Name: FnameFuncRun}, nil,
		getWrappedFuncOperate("FuncRun", "run", func(fc stf.Function) error { return fc.Run() }),
	)
}

func DeclareStfsFuncStart(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FnameFuncStart}, nil,
		getWrappedFuncOperate("FuncStart", "started", func(fc stf.Function) error { return fc.Start() }),
	)
}
func DeclareStfsFuncWait(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FnameFuncWait}, nil,
		getWrappedFuncOperate("FuncWait", "awaited", func(fc stf.Function) error { return fc.Wait() }),
	)
}

func DeclareStfsFuncTerminate(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FnameFuncTerminate}, nil,
		getWrappedFuncOperate("FuncTerminate", "terminated", func(fc stf.Function) error { fc.Terminate(); return nil }),
	)
}
