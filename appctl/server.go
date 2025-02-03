package appctl

import "github.com/t-beigbeder/otvl_qstf/stf"

type AppServer interface {
	AddIStream(id string) (IStream, error)
	GetOStream(id string) (OStream, error)
	GetFunction(id string, iss []IStream, oss []OStream) (FcServer, error)
}

type FcServer interface {
	GetFunction(id string) (stf.Function, error)
}
