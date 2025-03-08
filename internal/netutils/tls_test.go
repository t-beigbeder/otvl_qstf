package netutils

import (
	"context"
	"crypto/tls"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestSelfSigned(t *testing.T) {
	cert, err := SelfSigned("localhost")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{*cert}}
	srv := &http.Server{
		Addr:         "localhost:9443",
		TLSConfig:    cfg,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}
	go func() {
		_ = srv.ListenAndServeTLS("", "")
	}()
	defer func() { _ = srv.Shutdown(context.TODO()) }()
	time.Sleep(200 * time.Millisecond)
	hc := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	get, err := hc.Get("https://localhost:9443")
	if err != nil {
		t.Fatal(err)
	}
	if get.StatusCode != http.StatusNotFound {
		t.Fatalf("status code %d", get.StatusCode)
	}
}

func TestNewServerCert(t *testing.T) {
	pool, caCert, caPrik, err := NewCaCert()
	require.NoError(t, err)
	_, _, _ = pool, caCert, caPrik
	tlsCert, err := NewCert([]string{"0.0.0.0", "localhost"}, caCert, caPrik)
	require.NoError(t, err)
	_ = tlsCert

	cfg := &tls.Config{Certificates: []tls.Certificate{*tlsCert}}
	srv := &http.Server{
		Addr:         "0.0.0.0:9443",
		TLSConfig:    cfg,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}
	go func() {
		_ = srv.ListenAndServeTLS("", "")
	}()
	defer func() { _ = srv.Shutdown(context.TODO()) }()
	time.Sleep(200 * time.Millisecond)
	hc := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}
	get, err := hc.Get("https://localhost:9443")
	if err != nil {
		t.Fatal(err)
	}
	if get.StatusCode != http.StatusNotFound {
		t.Fatalf("status code %d", get.StatusCode)
	}
}
