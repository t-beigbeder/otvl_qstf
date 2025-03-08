package quic

import (
	"crypto/tls"
	"errors"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"time"
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
	// ALPN protocol names, nil is OK for HTTPS TLS handshake
	Alpns []string
	// sends keep alive packets if period set, see quic.Config
	KeepAlivePeriod time.Duration
	// TLS options
	TlsOptions
}

// GetConfig provides the TLS and QUIC configuration according to the given options
func GetConfig(qo *QuicOptions) (*tls.Config, *quic.Config, error) {
	var (
		certs []tls.Certificate
		tc    *tls.Config
		qc    *quic.Config
	)
	if qo.GenSelfSigned {
		cert, err := netutils.SelfSigned(qo.SelfSignedHost)
		if err != nil {
			return nil, nil, err
		}
		certs = []tls.Certificate{*cert}
	} else {
		if qo.TlsOptions.CertFile != "" || qo.TlsOptions.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(qo.TlsOptions.CertFile, qo.TlsOptions.KeyFile)
			if err != nil {
				return nil, nil, err
			}
			certs = []tls.Certificate{cert}
		} else if qo.IsServer {
			return nil, nil, errors.New("TLS server certificate is not provided")
		}
	}
	if !qo.IsServer && qo.InsecureSkipVerify {
		tc = &tls.Config{InsecureSkipVerify: true, NextProtos: qo.Alpns, Certificates: certs}
		qc = &quic.Config{KeepAlivePeriod: qo.KeepAlivePeriod}
	}
	if qo.IsServer {

	}
	qc = &quic.Config{}
	if qo.KeepAlivePeriod != 0 {
		qc.KeepAlivePeriod = qo.KeepAlivePeriod
	}
	return tc, qc, nil
}
