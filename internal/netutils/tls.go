package netutils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

// https://go.dev/src/crypto/tls/generate_cert.go
// /usr/local/go/src/crypto/tlsutils/generate_cert.go
// https://pkg.go.dev/crypto/tls#example-X509KeyPair

func SelfSigned(host string) (*tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"otvl"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	template.DNSNames = append(template.DNSNames, host)
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}
	certPem := bytes.Buffer{}
	err = pem.Encode(&certPem, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return nil, err
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	keyPem := bytes.Buffer{}
	if err := pem.Encode(&keyPem, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPem.Bytes(), keyPem.Bytes())
	return &cert, err
}

func GetUnsafeTlsConfigClient(alpn string) *tls.Config {
	var np []string
	if alpn != "" {
		np = []string{alpn}
	}
	return &tls.Config{ServerName: "unsafe-host", InsecureSkipVerify: true, Certificates: nil, NextProtos: np}
}
