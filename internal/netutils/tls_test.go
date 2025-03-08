package netutils

import (
	"context"
	"crypto/tls"
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
