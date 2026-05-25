package main

import (
	"crypto/tls"
	"encoding/json"
	"mist/util"
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

	// Dashboard metrics.
	startedAt       time.Time
	activeSessions  atomic.Int64
	acceptedConns   atomic.Int64

	// Server config snapshot for the dashboard.
	listenAddr string
	certType   string
	certName   string
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
		startedAt:         time.Now().UTC(),
	}
	return s
}

// SetConfigInfo stores server config metadata for the dashboard.
func (s *myServer) SetConfigInfo(listenAddr, certType, certName string) {
	s.listenAddr = listenAddr
	s.certType = certType
	s.certName = certName
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

// StatusJSON implements web.StatusProvider.
func (s *myServer) StatusJSON() ([]byte, error) {
	body := map[string]any{
		"version":              util.Version,
		"commit":               util.Commit,
		"date":                 util.Date,
		"started_at":           s.startedAt.Format(time.RFC3339),
		"updated_at":           time.Now().UTC().Format(time.RFC3339),
		"server":               s.listenAddr,
		"server_state":         "running",
		"active_connections":   s.activeSessions.Load(),
		"active_sessions":      s.activeSessions.Load(),
		"accepted_connections": s.acceptedConns.Load(),
		"listen":               s.listenAddr,
		"cert_type":            s.certType,
	}
	if s.certName != "" {
		body["cert_name"] = s.certName
	}
	return json.Marshal(body)
}

// ConfigJSON implements web.ConfigProvider.
func (s *myServer) ConfigJSON() ([]byte, error) {
	cfg := map[string]any{
		"type":               "server",
		"server":             s.listenAddr,
		"server_state":       "running",
		"listen":             s.listenAddr,
		"inbound":            "",
		"redirect_listen":    "",
		"min_idle_session":   0,
		"tls_min_version":    "",
		"insecure":           false,
		"cert_type":          s.certType,
		"cert_name":          s.certName,
		"max_streams":        s.maxStreams,
		"read_timeout":       s.readTimeout.String(),
		"keepalive":          s.keepaliveInterval.String(),
		"syn_rate_limit":     s.synRateLimit,
		"has_fallback":       s.fallbackAddr != "",
		"active_sessions":    s.activeSessions.Load(),
		"accepted_connections": s.acceptedConns.Load(),
	}
	if s.fallbackAddr != "" {
		cfg["fallback"] = s.fallbackAddr
	}
	return json.Marshal(cfg)
}
