package dns

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// Config holds DNS proxy configuration.
type Config struct {
	Listen   string
	Upstream []string
	Timeout  time.Duration
}

// Forwarder sends DNS queries to an upstream resolver.
type Forwarder interface {
	Forward(ctx context.Context, msg []byte) ([]byte, error)
}

// Stats holds DNS proxy statistics.
type Stats struct {
	QueriesForwarded uint64   `json:"queries_forwarded"`
	QueriesCached    uint64   `json:"queries_cached"`
	QueriesFailed    uint64   `json:"queries_failed"`
	CacheSize        int      `json:"cache_size"`
	Upstreams        []string `json:"upstreams"`
}

// Server is a DNS proxy that forwards queries through a configured Forwarder.
type Server struct {
	cfg       Config
	forwarder Forwarder
	cache     *Cache
	split     *SplitRouter
	server    *dns.Server

	mu    sync.Mutex
	stats Stats
	die   chan struct{}
}

// New creates a new DNS proxy server.
func New(cfg Config, forwarder Forwarder) *Server {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Server{
		cfg:       cfg,
		forwarder: forwarder,
		cache:     NewCache(512),
		split:     NewSplitRouter(),
		die:       make(chan struct{}),
	}
}

// Start begins serving DNS queries.
func (s *Server) Start() error {
	s.stats.Upstreams = s.cfg.Upstream

	s.server = &dns.Server{
		Addr:    s.cfg.Listen,
		Net:     "udp",
		Handler: dns.HandlerFunc(s.ServeDNS),
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			select {
			case <-s.die:
				return
			default:
				logrus.Errorf("DNS server: %v", err)
			}
		}
	}()

	logrus.Infof("DNS proxy listening on %s", s.cfg.Listen)
	return nil
}

// Stop shuts down the DNS proxy.
func (s *Server) Stop() error {
	close(s.die)
	if s.server != nil {
		return s.server.Shutdown()
	}
	return nil
}

// ServeDNS handles a DNS query.
func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		return
	}
	q := r.Question[0]
	key := cacheKey(q)

	// Check cache.
	if cached, ok := s.cache.Get(key); ok {
		cached.Id = r.Id
		s.incrCached()
		w.WriteMsg(cached)
		return
	}

	// Forward to upstream.
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Timeout)
	defer cancel()

	raw, err := r.Pack()
	if err != nil {
		s.incrFailed()
		return
	}

	resp, err := s.forwarder.Forward(ctx, raw)
	if err != nil {
		s.incrFailed()
		logrus.Debugf("DNS forward failed for %s: %v", q.Name, err)
		return
	}

	s.incrForwarded()

	msg := new(dns.Msg)
	if err := msg.Unpack(resp); err != nil {
		return
	}
	msg.Id = r.Id

	s.cache.Put(key, msg)
	w.WriteMsg(msg)
}

func (s *Server) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.CacheSize = s.cache.Len()
	return s.stats
}

// AddTunnelDomain adds a domain that must be resolved through the tunnel.
func (s *Server) AddTunnelDomain(pattern string) {
	s.split.AddTunnelDomain(pattern)
}

// AddDirectDomain adds a domain that should be resolved directly.
func (s *Server) AddDirectDomain(pattern string) {
	s.split.AddDirectDomain(pattern)
}

func (s *Server) incrCached() {
	s.mu.Lock()
	s.stats.QueriesCached++
	s.mu.Unlock()
}

func (s *Server) incrForwarded() {
	s.mu.Lock()
	s.stats.QueriesForwarded++
	s.mu.Unlock()
}

func (s *Server) incrFailed() {
	s.mu.Lock()
	s.stats.QueriesFailed++
	s.mu.Unlock()
}

func cacheKey(q dns.Question) string {
	return fmt.Sprintf("%s:%d:%d", q.Name, q.Qtype, q.Qclass)
}
