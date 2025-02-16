package appctl

import (
	"context"
	"errors"
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"os/exec"
)

type CommandExitStatus struct {
	ExitCode int `json:"exit-code"`
}

type lcStartWait struct {
	cmd *exec.Cmd
}

var _ stf.StartWaiter = &lcStartWait{}

func (sw *lcStartWait) Start(ctx context.Context) error {
	fc := CurrentFunction(ctx)
	if len(fc.GetInStreams()) != 1 || len(fc.GetOutStreams()) != 2 {
		return fmt.Errorf("invalid in/out streams count %d/%d", len(fc.GetInStreams()), len(fc.GetOutStreams()))
	}
	aInPl, ok := CurrentValues(ctx)["in-pl"]
	if !ok {
		return fmt.Errorf("in-pl not found in context")
	}
	cs, ok := aInPl.(*stf.CommandSpec)
	if !ok {
		return fmt.Errorf("in-pl type %T expected %T", aInPl, &stf.CommandSpec{})
	}
	sw.cmd = exec.CommandContext(ctx, cs.Cmd, cs.Args...)
	sw.cmd.Env = cs.Env
	sw.cmd.Dir = cs.Dir
	sw.cmd.Stdin = fc.GetInStreams()[0]
	sw.cmd.Stdout = fc.GetOutStreams()[1]
	sw.cmd.Stderr = fc.GetOutStreams()[2]
	err := sw.cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func (sw *lcStartWait) Wait(ctx context.Context) error {
	err := sw.cmd.Wait()
	if err != nil {
		ee := &exec.ExitError{}
		if errors.As(err, &ee) {
			CurrentValues(ctx)["out-pl"] = &CommandExitStatus{ExitCode: ee.ExitCode()}
			return nil
		}
		return err
	}
	return nil
}

func LocalCommandFuncDeclarer(fName, isName, osName, esName string) func(*FunctionCatalog) error {
	return func(cat *FunctionCatalog) error {
		return cat.DeclareFunction(
			FunctionDesc{
				Name: fName,
				IStreams: []IStreamDesc{
					{StreamDesc: StreamDesc{Name: isName}},
				},
				OStreams: []OStreamDesc{
					{StreamDesc: StreamDesc{Name: osName}},
					{StreamDesc: StreamDesc{Name: esName}},
				},
				Wrapper: WrapperDesc{InMarshaller: MarshalJson, OutMarshaller: MarshalJson},
			},
			&stf.WrappedFunction{
				InputTemplate:  FactoryFor[stf.CommandSpec],
				OutputTemplate: FactoryFor[CommandExitStatus],
			},
			&lcStartWait{},
			&StreamHandlers{
				ihs: []IStreamHandler{{}},
				ohs: []OStreamHandler{{}, {}},
			})
	}
}
