package quic

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
)

// TlsOptions can be used to configure TLS on the client or the server
type TlsOptions struct {
	// InsecureSkipVerify skips server certificate check
	InsecureSkipVerify bool

	// SelfSigned enables to generate a self-signed key pair certificate
	GenSelfSigned bool

	// SelfSignedHost is the host for which the self-signed certificate is generated, defaults to localhost
	SelfSignedHost string

	// CertFile certificate public key file
	CertFile string

	// KeyFile certificate private key file
	KeyFile string
}

// QuicOptions can be used to configure QUIC on the client or the server
type QuicOptions struct {
	// Some options are client or server specific
	IsServer bool
	//
	TlsOptions
	Alpns []string
}

// GetConfig provides the TLS and QUIC configuration according to the given options
func GetConfig(qo *QuicOptions) (tc *tls.Config, qc *quic.Config, err error) {
	if !qo.IsServer && qo.InsecureSkipVerify {
		tc = netutils.GetUnsafeTlsConfigClient(qo.Alpns)
		qc = &quic.Config{}
	}
	return
}
