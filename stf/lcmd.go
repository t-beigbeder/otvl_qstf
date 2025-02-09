package stf

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

type CommandSpec struct {
	Cmd  string   `json:"cmd,omitempty"`
	Args []string `json:"args,omitempty"`
	Env  []string `json:"env,omitempty"`
	Dir  string   `json:"dir,omitempty"`
}

type lcStartWait struct {
	cmd    *exec.Cmd
	is     InStream
	os, es OutStream
}

var _ StartWaiter = &lcStartWait{}

func (sw *lcStartWait) Start(_ context.Context) error {
	return sw.cmd.Start()
}

func (sw *lcStartWait) Wait(_ context.Context) error {
	err := sw.cmd.Wait()
	if err != nil {
		return err
	}
	sw.is.Terminate()
	sw.os.Terminate()
	sw.es.Terminate()
	return nil
}

func NewLocalCommand(
	ctx context.Context, cs CommandSpec,
	stdin io.Reader, stdout, stderr io.Writer,
	opts ...FcOption,
) (Function, error) {
	cmd := exec.CommandContext(ctx, cs.Cmd, cs.Args...)
	cmd.Env = cs.Env
	cmd.Dir = cs.Dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	sw := &lcStartWait{cmd: cmd}
	fc, err := NewFunction(ctx, sw, opts...)
	if err != nil {
		return nil, err
	}
	if fc.Options().Terminable {
		return nil, errors.New("a local command function cannot be set terminable")
	}
	if sw.is, err = fc.AddInStream(stdin, IstId("stdin")); err != nil {
		return nil, err
	}
	if sw.os, err = fc.AddOutStream(stdout, OstId("stdout")); err != nil {
		return nil, err
	}
	if sw.es, err = fc.AddOutStream(stderr, OstId("stderr")); err != nil {
		return nil, err
	}
	return fc, nil
}
