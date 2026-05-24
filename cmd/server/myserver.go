package main

import (
	"crypto/tls"
	"time"
)

type myServer struct {
	tlsConfig         *tls.Config
	fallbackAddr      string
	maxStreams        int
	readTimeout       time.Duration
	keepaliveInterval time.Duration
	synRateLimit      int
	passwordHash      []byte
}

func NewMyServer(tlsConfig *tls.Config, fallbackAddr string, maxStreams int, readTimeout, keepaliveInterval time.Duration, synRateLimit int, passwordHash []byte) *myServer {
	s := &myServer{
		tlsConfig:         tlsConfig,
		fallbackAddr:      fallbackAddr,
		maxStreams:        maxStreams,
		readTimeout:       readTimeout,
		keepaliveInterval: keepaliveInterval,
		synRateLimit:      synRateLimit,
		passwordHash:      passwordHash,
	}
	return s
}
