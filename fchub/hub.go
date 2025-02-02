package fchub

import "github.com/t-beigbeder/otvl_qstf/stf"

type FcHub interface {
	AddIStream() (IStream, error)
	GetOStream() (OStream, error)
	AddFunction(stf.Function, []IStream, []OStream) error
}
