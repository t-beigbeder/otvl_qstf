package quic

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
