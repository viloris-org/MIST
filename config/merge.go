package config

// CLIOverrides holds CLI flag values that should override config file settings.
// A field is considered "set" if explicitly provided on the command line.
type CLIOverrides struct {
	Server         string // -s
	Password       string // -p
	SNI            string // -sni
	TLSCertSHA256  string // -tls-cert-sha256
	Insecure       bool   // -insecure
	TLSMinVersion  string // -tls-min-version
	MinIdleSession int    // -m
	Listen         string // -l
	Inbound        string // -inbound
	RedirectListen string // -redirect-listen
	LogFormat      string // -log-format
	LogFile        string // -log-file
	Tun            bool   // -tun
	TunName        string // -tun-name
	TunMTU         int    // -tun-mtu
	TunAddr        string // -tun-addr
	TunDNS         string // -tun-dns
	TunRoutes      string // -tun-routes
	DNS            bool   // -dns
	DNSListen      string // -dns-listen
	DNSUpstream    string // -dns-upstream
	Web            bool   // -web
	WebListen      string // -web-listen
	StatusJSON     string // -status-json
}

// ApplyCLIOverrides merges CLI flags into the config file values.
// CLI values take precedence when explicitly set.
func (c *ClientConfig) ApplyCLIOverrides(cli *CLIOverrides) {
	if cli.Server != "" {
		c.Server = cli.Server
	}
	if cli.Password != "" {
		c.Password = cli.Password
	}
	if cli.SNI != "" {
		c.TLS.SNI = cli.SNI
	}
	if cli.TLSCertSHA256 != "" {
		c.TLS.CertSHA256 = cli.TLSCertSHA256
	}
	if cli.Insecure {
		c.TLS.Insecure = true
	}
	if cli.TLSMinVersion != "" {
		c.TLS.MinVersion = cli.TLSMinVersion
	}
	if cli.MinIdleSession != 0 {
		c.TLS.MinIdleSession = cli.MinIdleSession
	}
	if cli.Listen != "" {
		c.Inbound.Listen = cli.Listen
	}
	if cli.Inbound != "" {
		c.Inbound.Types = nil // clear defaults
		for _, t := range splitCSV(cli.Inbound) {
			if t == "socks" || t == "http" {
				if !contains(c.Inbound.Types, "socks") && !contains(c.Inbound.Types, "http") {
					c.Inbound.Types = append(c.Inbound.Types, t)
				}
			} else if t == "redirect" {
				c.Inbound.Types = append(c.Inbound.Types, t)
			}
		}
	}
	if cli.RedirectListen != "" {
		c.Inbound.RedirectListen = cli.RedirectListen
	}
	if cli.LogFormat != "" {
		c.Log.Format = cli.LogFormat
	}
	if cli.LogFile != "" {
		c.Log.File = cli.LogFile
	}

	// TUN overrides
	if cli.Tun {
		c.Tun.Enabled = true
	}
	if cli.TunName != "" {
		c.Tun.Name = cli.TunName
	}
	if cli.TunMTU != 0 {
		c.Tun.MTU = cli.TunMTU
	}
	if cli.TunAddr != "" {
		c.Tun.Address = cli.TunAddr
	}
	if cli.TunDNS != "" {
		c.Tun.DNS = filterEmpty(splitCSV(cli.TunDNS))
	}
	if cli.TunRoutes != "" {
		c.Tun.Routes = filterEmpty(splitCSV(cli.TunRoutes))
	}

	// DNS overrides
	if cli.DNS {
		c.DNS.Enabled = true
	}
	if cli.DNSListen != "" {
		c.DNS.Listen = cli.DNSListen
	}
	if cli.DNSUpstream != "" {
		c.DNS.Upstream = filterEmpty(splitCSV(cli.DNSUpstream))
	}

	// Web overrides
	if cli.Web {
		c.Web.Enabled = true
	}
	if cli.WebListen != "" {
		c.Web.Listen = cli.WebListen
	}
}

func splitCSV(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func filterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, item := range ss {
		if item == s {
			return true
		}
	}
	return false
}
