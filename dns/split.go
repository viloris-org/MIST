package dns

import "strings"

// SplitRouter determines whether a DNS domain should be resolved through the
// tunnel or directly.
type SplitRouter struct {
	tunnelDomains []domainPattern
	directDomains []domainPattern
}

type domainPattern struct {
	pattern string
	suffix  bool // true if pattern should match as a suffix (e.g., "*.example.com")
}

// NewSplitRouter creates a new split router with no rules.
func NewSplitRouter() *SplitRouter {
	return &SplitRouter{}
}

// AddTunnelDomain adds a domain pattern that must be resolved through the tunnel.
// Patterns support "*." prefix for suffix matching (e.g., "*.internal" matches
// "foo.internal" and "bar.internal").
func (r *SplitRouter) AddTunnelDomain(pattern string) {
	r.tunnelDomains = append(r.tunnelDomains, parsePattern(pattern))
}

// AddDirectDomain adds a domain pattern that should be resolved directly.
func (r *SplitRouter) AddDirectDomain(pattern string) {
	r.directDomains = append(r.directDomains, parsePattern(pattern))
}

// ShouldTunnel returns true if the domain should be routed through the tunnel.
func (r *SplitRouter) ShouldTunnel(domain string) bool {
	domain = normalizeDomain(domain)

	// Direct rules take priority (explicit bypass).
	for _, p := range r.directDomains {
		if p.match(domain) {
			return false
		}
	}
	// Tunnel rules.
	for _, p := range r.tunnelDomains {
		if p.match(domain) {
			return true
		}
	}
	// Default: tunnel all unless a direct rule matched.
	return len(r.directDomains) == 0
}

func parsePattern(pattern string) domainPattern {
	return domainPattern{
		pattern: normalizeDomain(strings.TrimPrefix(pattern, "*.")),
		suffix:  strings.HasPrefix(pattern, "*."),
	}
}

func (p domainPattern) match(domain string) bool {
	if p.suffix {
		return domain == p.pattern || strings.HasSuffix(domain, "."+p.pattern)
	}
	return domain == p.pattern
}

func normalizeDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}
