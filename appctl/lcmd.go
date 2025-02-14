package appctl

import (
	"context"
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"os/exec"
)

type lcStartWait struct {
	cmd *exec.Cmd
}

var _ stf.StartWaiter = &lcStartWait{}

func (sw *lcStartWait) Start(ctx context.Context) error {
	fc := CurrentFunction(ctx)
	if len(fc.GetInStreams()) != 1 || len(fc.GetOutStreams()) != 2 {
		return fmt.Errorf("invalid in/out streams count %d/%d", len(fc.GetInStreams()), len(fc.GetOutStreams()))
	}
	//sw.cmd = exec.CommandContext(ctx)
	//return sw.cmd.Start()
	return nil
}

func (sw *lcStartWait) Wait(_ context.Context) error {
	//err := sw.cmd.Wait()
	//if err != nil {
	//	return err
	//}
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
			},
			nil, &lcStartWait{}, nil)
	}
}
