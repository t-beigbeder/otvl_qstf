package appctl

import "github.com/t-beigbeder/otvl_qstf/stf"

type AppClient interface {
	AddIStream(id string) (OStream, error)
	GetOStream(id string) (IStream, error)
	GetFunction(id string, iss []OStream, oss []IStream) (FcClient, error)
}

type FcClient interface {
	Run() error
	Start() error
	Wait() error
	Terminate()
	State() stf.FunctionState
	Options() stf.FcOptions
	Error() error
}
