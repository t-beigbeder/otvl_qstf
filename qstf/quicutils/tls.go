package quicutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"path/filepath"
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

	// CACertFile certificate public key file
	CACertFile string

	// CertFile certificate public key file
	CertFile string

	// KeyFile certificate private key file
	KeyFile string

	// ClientAuth requires server to check mTLS client certificate
	ClientAuth bool

	// VerifyPeerCertificate configures the server for checking client certificates
	VerifyPeerCertificate func([][]*x509.Certificate) error
}

// QuicOptions can be used to configure QUIC on the client or the server
type QuicOptions struct {
	// Some options are client or server specific
	IsServer bool
	// ALPN protocol names, nil is OK for HTTPS TLS handshake
	Alpns []string
	// sends keep alive packets if period set, see quic.Config
	KeepAlivePeriod time.Duration
	// Tracer
	Tracer func(context.Context, logging.Perspective, quic.ConnectionID) *logging.ConnectionTracer

	// TLS options
	TlsOptions
}

// GetConfig provides the TLS and QUIC configuration according to the given options
func GetConfig(qo *QuicOptions) (*tls.Config, *quic.Config, error) {
	var (
		certs    []tls.Certificate
		certPool *x509.CertPool
		tc       *tls.Config
		qc       *quic.Config
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
		if qo.TlsOptions.CACertFile != "" {
			caPEM, err := common.LoadFile(filepath.Join(qo.TlsOptions.CACertFile))
			if err != nil {
				return nil, nil, err
			}
			certPool = x509.NewCertPool()
			certPool.AppendCertsFromPEM(caPEM)
		}
	}
	if !qo.IsServer {
		if qo.InsecureSkipVerify {
			tc = &tls.Config{InsecureSkipVerify: true, NextProtos: qo.Alpns, Certificates: certs}
		} else {
			tc = &tls.Config{NextProtos: qo.Alpns, Certificates: certs, RootCAs: certPool}
		}
	} else {
		tc = &tls.Config{NextProtos: qo.Alpns, Certificates: certs, RootCAs: certPool}
		if qo.ClientAuth {
			tc.ClientAuth = tls.RequireAndVerifyClientCert
			tc.ClientCAs = certPool
		}
		if qo.VerifyPeerCertificate != nil {
			tc.VerifyPeerCertificate = func(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
				return qo.VerifyPeerCertificate(verifiedChains)
			}
		}
	}
	qc = &quic.Config{KeepAlivePeriod: qo.KeepAlivePeriod, Tracer: qo.Tracer}
	return tc, qc, nil
}
