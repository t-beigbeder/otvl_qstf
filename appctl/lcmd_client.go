package appctl

import (
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
	if osti, err = ac.AddIStream(spec.IsName); err != nil {
		return
	}
	if isto, err = ac.GetOStream(spec.OsName); err != nil {
		return
	}
	if iste, err = ac.GetOStream(spec.EsName); err != nil {
		return
	}
	if fc, err = ac.NewFunction(spec.FName, spec.FId); err != nil {
		return
	}
	if err = fc.AddOStream(osti); err != nil {
		return
	}
	if err = fc.AddIStream(isto); err != nil {
		return
	}
	if err = fc.AddIStream(iste); err != nil {
		return
	}
	return
}
