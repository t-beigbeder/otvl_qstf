package netutils

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"net"
	"strings"
)

func GetQuicListener(addr string, tc *tls.Config, qc *quic.Config) (*quic.Listener, string, string, error) {
	ip, port, err := GetIPPort(addr)
	if err != nil {
		return nil, "", "", err
	}
	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: ip, Port: port})
	if err != nil {
		return nil, "", "", err
	}
	lst, err := quic.Listen(udpConn, tc, qc)
	if err != nil {
		return nil, "", "", err
	}
	las := strings.Split(lst.Addr().String(), ":")
	return lst, las[0], las[len(las)-1], nil
}
