package main

import (
	"crypto/tls"
	"sync/atomic"
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

	activeSessions  atomic.Int64
	acceptedConns   atomic.Int64
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

// SessionAccepted increments the active session and total connection counters.
func (s *myServer) SessionAccepted() {
	s.activeSessions.Add(1)
	s.acceptedConns.Add(1)
}

// SessionClosed decrements the active session counter.
func (s *myServer) SessionClosed() {
	s.activeSessions.Add(-1)
}

