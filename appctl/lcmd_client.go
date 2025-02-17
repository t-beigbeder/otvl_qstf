package appctl

import (
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"io"
)

type LocalCommandClientSpec struct {
	FName  string
	FId    string
	IsName string
	OsName string
	EsName string
}

func NewLocalCommandClient(ac AppClient, spec LocalCommandClientSpec) (fc FcClient, stdin io.Writer, stdout io.Reader, stderr io.Reader, err error) {
	var (
		osti OStream
		isto IStream
		iste IStream
	)
	if fc, err = ac.NewFunction(spec.FName, spec.FId); err != nil {
		return
	}
	stdid := func(id, kind string) string {
		if id != "" {
			return id
		}
		return fmt.Sprintf("%s/%s", fc.GetId(), kind)
	}
	if osti, err = ac.AddIStream(stdid(spec.IsName, "in")); err != nil {
		return
	}
	if isto, err = ac.GetOStream(stdid(spec.OsName, "out")); err != nil {
		return
	}
	if iste, err = ac.GetOStream(stdid(spec.EsName, "err")); err != nil {
		return
	}
	if err = fc.AddIStream(osti); err != nil {
		return
	}
	if err = fc.AddOStream(isto); err != nil {
		return
	}
	if err = fc.AddOStream(iste); err != nil {
		return
	}
	stdin, stdout, stderr = osti, isto, iste
	return
}

func StartNewLocalCommand(ac AppClient, cls LocalCommandClientSpec, cms *stf.CommandSpec) (fc FcClient, stdin io.Writer, stdout io.Reader, stderr io.Reader, err error) {
	fc, stdin, stdout, stderr, err = NewLocalCommandClient(ac, cls)
	if err != nil {
		return
	}
	err = fc.StartWith(MarshalJson, cms)
	if err != nil {
		return
	}
	return
}

func WaitForLocalCommand(fc FcClient) (exitCode int, err error) {
	es := CommandExitStatus{}
	if err := fc.WaitWith(MarshalJson, &es); err != nil {
		return 0, err
	}
	return es.ExitCode, nil
}
