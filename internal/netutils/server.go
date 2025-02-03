package netutils

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"log/slog"
	"net"
	"strings"
)

func GetQuicListener(addr string, cert *tls.Certificate, alpn string, logger *slog.Logger) (*quic.Listener, string, string, error) {
	ip, port, err := GetIPPort(addr)
	if err != nil {
		return nil, "", "", err
	}
	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: ip, Port: port})
	if err != nil {
		return nil, "", "", err
	}
	qc := quic.Config{Tracer: qlog.DefaultConnectionTracer}
	tc := tls.Config{
		Certificates: []tls.Certificate{*cert},
		NextProtos:   []string{alpn},
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			logger.Info("connection", "ServerName", info.ServerName, "SupportedProtos", info.SupportedProtos)
			return nil, nil
		},
	}
	lst, err := quic.Listen(udpConn, &tc, &qc)
	if err != nil {
		return nil, "", "", err
	}
	las := strings.Split(lst.Addr().String(), ":")
	return lst, las[0], las[len(las)-1], nil
}
