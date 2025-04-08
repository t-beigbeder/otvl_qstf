package qstf

import (
	"context"
	"github.com/quic-go/quic-go"
	"log"
	"sync"
)

type serverId struct {
	addr   string
	hostId string
}

type ClientHost struct {
	ctx            context.Context
	logger         *log.Logger
	mx             sync.Mutex
	HostId         string
	serverRegistry map[serverId]quic.Connection
}
