package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mist/config"
	"mist/dns"
	"mist/mistclient"
	"mist/tun"
	"mist/util"

	"MistCore/common/buf"
	"MistCore/common/bufio"
	M "MistCore/common/metadata"
	"MistCore/common/uot"

	"github.com/sirupsen/logrus"
)

type clientConfig struct {
	configPath      string
	listen          string
	redirectListen  string
	inbound         string
	serverAddr      string
	sni             string
	password        string
	tlsCertSha256   string
	insecure        bool
	minIdleSession  int
	tlsMinVersion   string
	transport       string
	tlsProfile      string
	trafficProfile  string
	showVersion     bool
	showVersionJSON bool
	check           bool
	statusJSON      string
	logFormat       string
	logFile         string

	// New features
	tunEnabled bool
	tunName    string
	tunMTU     int
	tunAddr    string
	tunDNS     string
	tunRoutes  string

	dnsEnabled  bool
	dnsListen   string
	dnsUpstream string
}

type inboundSet struct {
	socksHTTP bool
	redirect  bool
	tun       bool
	dns       bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		logrus.Fatalln(err)
	}
}

func run(args []string) error {
	cfg, err := parseClientConfig(args)
	if err != nil {
		return err
	}

	// Load config file if specified.
	if cfg.configPath != "" {
		if err := loadConfigFile(&cfg); err != nil {
			return err
		}
	}

	if cfg.showVersionJSON {
		return printVersionJSON(os.Stdout)
	}
	if cfg.showVersion {
		fmt.Fprintf(os.Stdout, "mist-client %s commit=%s date=%s\n", util.Version, util.Commit, util.Date)
		return nil
	}

	if err := configureLogging(cfg.logFormat, cfg.logFile); err != nil {
		return err
	}

	opts, err := cfg.clientOptions()
	if err != nil {
		return err
	}
	inbounds, err := cfg.parseInboundSet()
	if err != nil {
		return err
	}
	if cfg.check {
		if _, err := opts.TLSConfig(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "ok")
		return nil
	}

	if path := strings.TrimSpace(os.Getenv("TLS_KEY_LOG")); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("open TLS_KEY_LOG: %w", err)
		}
		defer f.Close()
		opts.KeyLogWriter = f
	}

	client, err := mistclient.NewClient(opts)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer client.Close()

	// Global shared status.
	globalStatus := newGlobalStatus(cfg)
	globalStatus.SetClient(client)

	status := newStatusReporter(cfg.statusJSON, cfg.serverAddr, cfg.listen, cfg.redirectListen, cfg.inbound, client, globalStatus)
	status.SetServerState("running")
	status.Write()
	status.Start()
	defer status.Stop()

	logrus.Infoln("[Client]", util.ProgramVersionName)
	logrus.Infoln("[Client] inbound", cfg.inbound, "=>", cfg.serverAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 5)

	// TUN device.
	if inbounds.tun {
		tunDev, err := startTUN(ctx, cfg, client)
		if err != nil {
			return fmt.Errorf("start TUN: %w", err)
		}
		defer tunDev.Close()
		globalStatus.SetTUN(tunDev)
		logrus.Infoln("[Client] tun", tunDev.Name(), cfg.tunAddr, "=>", cfg.serverAddr)
	}

	// DNS proxy.
	if inbounds.dns {
		dnsSrv, err := startDNS(ctx, cfg, client)
		if err != nil {
			return fmt.Errorf("start DNS: %w", err)
		}
		defer dnsSrv.Stop()
		globalStatus.SetDNS(dnsSrv)
	}

	// SOCKS/HTTP listener.
	if inbounds.socksHTTP {
		listener, err := net.Listen("tcp", cfg.listen)
		if err != nil {
			return fmt.Errorf("listen socks/http tcp: %w", err)
		}
		logrus.Infoln("[Client] socks5/http", cfg.listen, "=>", cfg.serverAddr)
		go func() {
			errCh <- acceptLoop(ctx, listener, status, func(conn net.Conn) {
				handleTcpConnection(ctx, conn, client)
			})
		}()
	}

	// Redirect listener.
	if inbounds.redirect {
		listener, err := net.Listen("tcp", cfg.redirectListen)
		if err != nil {
			return fmt.Errorf("listen redirect tcp: %w", err)
		}
		logrus.Infoln("[Client] redirect", cfg.redirectListen, "=>", cfg.serverAddr)
		go func() {
			errCh <- acceptLoop(ctx, listener, status, func(conn net.Conn) {
				handleRedirectConnection(ctx, conn, client)
			})
		}()
	}

	return <-errCh
}

func loadConfigFile(cfg *clientConfig) error {
	fileCfg, err := config.DecodeFile(cfg.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	fileCfg.SetDefaults()

	cli := &config.CLIOverrides{
		Server:         cfg.serverAddr,
		Password:       cfg.password,
		SNI:            cfg.sni,
		TLSCertSHA256:  cfg.tlsCertSha256,
		Insecure:       cfg.insecure,
		TLSMinVersion:  cfg.tlsMinVersion,
		MinIdleSession: cfg.minIdleSession,
		Transport:      cfg.transport,
		TLSProfile:     cfg.tlsProfile,
		TrafficProfile: cfg.trafficProfile,
		Listen:         cfg.listen,
		Inbound:        cfg.inbound,
		RedirectListen: cfg.redirectListen,
		LogFormat:      cfg.logFormat,
		LogFile:        cfg.logFile,
		Tun:            cfg.tunEnabled,
		TunName:        cfg.tunName,
		TunMTU:         cfg.tunMTU,
		TunAddr:        cfg.tunAddr,
		TunDNS:         cfg.tunDNS,
		TunRoutes:      cfg.tunRoutes,
		DNS:            cfg.dnsEnabled,
		DNSListen:      cfg.dnsListen,
		DNSUpstream:    cfg.dnsUpstream,
		StatusJSON:     cfg.statusJSON,
	}
	fileCfg.ApplyCLIOverrides(cli)

	// Copy back merged config to clientConfig.
	cfg.serverAddr = fileCfg.Server
	cfg.password = fileCfg.Password
	cfg.sni = fileCfg.TLS.SNI
	cfg.tlsCertSha256 = fileCfg.TLS.CertSHA256
	cfg.insecure = fileCfg.TLS.Insecure
	cfg.tlsMinVersion = fileCfg.TLS.MinVersion
	cfg.minIdleSession = fileCfg.TLS.MinIdleSession
	cfg.transport = fileCfg.TLS.Transport
	cfg.tlsProfile = fileCfg.TLS.TLSProfile
	cfg.trafficProfile = fileCfg.TLS.TrafficProfile
	cfg.listen = fileCfg.Inbound.Listen
	cfg.inbound = strings.Join(fileCfg.Inbound.Types, ",")
	cfg.redirectListen = fileCfg.Inbound.RedirectListen
	cfg.logFormat = fileCfg.Log.Format
	cfg.logFile = fileCfg.Log.File
	cfg.tunEnabled = fileCfg.Tun.Enabled
	cfg.tunName = fileCfg.Tun.Name
	cfg.tunMTU = fileCfg.Tun.MTU
	cfg.tunAddr = fileCfg.Tun.Address
	cfg.tunDNS = strings.Join(fileCfg.Tun.DNS, ",")
	cfg.tunRoutes = strings.Join(fileCfg.Tun.Routes, ",")
	cfg.dnsEnabled = fileCfg.DNS.Enabled
	cfg.dnsListen = fileCfg.DNS.Listen
	cfg.dnsUpstream = strings.Join(fileCfg.DNS.Upstream, ",")
	return nil
}

func (cfg clientConfig) parseInboundSet() (inboundSet, error) {
	var set inboundSet
	for _, item := range strings.Split(cfg.inbound, ",") {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "":
		case "socks", "http":
			set.socksHTTP = true
		case "redirect":
			set.redirect = true
		case "tun":
			set.tun = true
		case "dns":
			set.dns = true
		default:
			return set, fmt.Errorf("unsupported inbound %q", item)
		}
	}
	// New modes enabled by flags even if not in -inbound list.
	if cfg.tunEnabled {
		set.tun = true
	}
	if cfg.dnsEnabled {
		set.dns = true
	}
	if !set.socksHTTP && !set.redirect && !set.tun && !set.dns {
		return set, fmt.Errorf("at least one inbound is required")
	}
	return set, nil
}

// --- TUN ---

type tunHandler struct {
	client *mistclient.Client
}

func (h *tunHandler) HandleTCP(ctx context.Context, conn net.Conn, dest netip.AddrPort) error {
	metadata := M.Metadata{
		Destination: M.SocksaddrFrom(dest.Addr(), dest.Port()),
	}
	return h.client.NewConnection(ctx, conn, metadata)
}

func (h *tunHandler) HandleUDP(ctx context.Context, conn tun.UDPConn, dest netip.AddrPort) error {
	defer conn.Close()

	stream, err := h.client.DialStream(ctx, uot.RequestDestination(2))
	if err != nil {
		return err
	}
	defer stream.Close()

	uotConn := uot.NewLazyConn(stream, uot.Request{
		Destination: M.SocksaddrFrom(dest.Addr(), dest.Port()),
	})
	defer uotConn.Close()

	return bufio.CopyPacketConn(ctx, newUDPPacketConn(conn), uotConn)
}

// udpPacketConn adapts tun.UDPConn to network.PacketConn.
type udpPacketConn struct {
	conn      tun.UDPConn
	localAddr net.Addr
}

func newUDPPacketConn(conn tun.UDPConn) *udpPacketConn {
	return &udpPacketConn{
		conn:      conn,
		localAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0},
	}
}

func (c *udpPacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	data, remote, err := c.conn.ReadPacket()
	if err != nil {
		return M.Socksaddr{}, err
	}
	buffer.Reset()
	buffer.Write(data)
	return M.SocksaddrFrom(remote.Addr(), remote.Port()), nil
}

func (c *udpPacketConn) WritePacket(buffer *buf.Buffer, dest M.Socksaddr) error {
	return c.conn.WritePacket(buffer.Bytes(), netip.AddrPortFrom(dest.Addr, dest.Port))
}

func (c *udpPacketConn) Close() error                       { return c.conn.Close() }
func (c *udpPacketConn) LocalAddr() net.Addr                { return c.localAddr }
func (c *udpPacketConn) SetDeadline(t time.Time) error      { return nil }
func (c *udpPacketConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *udpPacketConn) SetWriteDeadline(t time.Time) error { return nil }

func startTUN(ctx context.Context, cfg clientConfig, client *mistclient.Client) (*tun.Device, error) {
	addr, err := netip.ParsePrefix(cfg.tunAddr)
	if err != nil {
		return nil, fmt.Errorf("parse tun address %q: %w", cfg.tunAddr, err)
	}

	var dnsAddrs []netip.Addr
	for _, s := range splitCSV(cfg.tunDNS) {
		if a, err := netip.ParseAddr(s); err == nil {
			dnsAddrs = append(dnsAddrs, a)
		}
	}

	var routes []netip.Prefix
	for _, s := range splitCSV(cfg.tunRoutes) {
		if p, err := netip.ParsePrefix(s); err == nil {
			routes = append(routes, p)
		}
	}
	if len(routes) == 0 {
		// Default: route everything through tunnel.
		routes = append(routes, netip.MustParsePrefix("0.0.0.0/0"))
	}

	tunCfg := tun.Config{
		Name:    cfg.tunName,
		MTU:     cfg.tunMTU,
		Address: addr,
		DNS:     dnsAddrs,
		Routes:  routes,
	}

	handler := &tunHandler{client: client}
	dev, err := tun.Create(tunCfg, handler)
	if err != nil {
		return nil, err
	}
	if err := dev.Start(); err != nil {
		dev.Close()
		return nil, err
	}
	return dev, nil
}

// --- DNS ---

type dnsForwarder struct {
	client   *mistclient.Client
	upstream string
}

func (f *dnsForwarder) Forward(ctx context.Context, msg []byte) ([]byte, error) {
	stream, err := f.client.DialStream(ctx, uot.RequestDestination(2))
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	dest := M.ParseSocksaddrHostPortStr(f.upstream, "53")
	uotConn := uot.NewLazyConn(stream, uot.Request{Destination: dest})
	defer uotConn.Close()

	if _, err := uotConn.Write(msg); err != nil {
		return nil, err
	}

	resp := make([]byte, 1500)
	n, err := uotConn.Read(resp)
	if err != nil {
		return nil, err
	}
	return resp[:n], nil
}

func startDNS(ctx context.Context, cfg clientConfig, client *mistclient.Client) (*dns.Server, error) {
	upstream := cfg.dnsUpstream
	if upstream == "" {
		upstream = "1.1.1.1"
	}

	dnsCfg := dns.Config{
		Listen:   cfg.dnsListen,
		Upstream: splitCSV(upstream),
		Timeout:  5 * time.Second,
	}

	forwarder := &dnsForwarder{
		client:   client,
		upstream: dnsCfg.Upstream[0],
	}

	srv := dns.New(dnsCfg, forwarder)
	if err := srv.Start(); err != nil {
		return nil, err
	}
	return srv, nil
}

// --- Global status (shared between statusReporter and status file) ---

type globalStatus struct {
	startedAt time.Time

	mu            sync.Mutex
	server        string
	serverState   string
	lastError     string
	activeConns   atomic.Int64
	acceptedConns atomic.Int64
	client        *mistclient.Client
	tun           *tun.Device
	dns           *dns.Server
	cfg           clientConfig
}

func newGlobalStatus(cfg clientConfig) *globalStatus {
	return &globalStatus{
		startedAt:   time.Now().UTC(),
		server:      cfg.serverAddr,
		serverState: "starting",
		cfg:         cfg,
	}
}

func (gs *globalStatus) SetClient(c *mistclient.Client) {
	gs.client = c
}

func (gs *globalStatus) SetTUN(d *tun.Device) {
	gs.tun = d
}

func (gs *globalStatus) SetDNS(s *dns.Server) {
	gs.dns = s
}

func (gs *globalStatus) SetServerState(s string) {
	gs.mu.Lock()
	gs.serverState = s
	gs.mu.Unlock()
}

func (gs *globalStatus) SetError(err error) {
	gs.mu.Lock()
	gs.lastError = err.Error()
	gs.mu.Unlock()
}

func (gs *globalStatus) StatusJSON() ([]byte, error) {
	gs.mu.Lock()
	st := gs.serverState
	le := gs.lastError
	gs.mu.Unlock()

	body := map[string]any{
		"version":              util.Version,
		"commit":               util.Commit,
		"date":                 util.Date,
		"server":               gs.server,
		"server_state":         st,
		"started_at":           gs.startedAt.Format(time.RFC3339),
		"updated_at":           time.Now().UTC().Format(time.RFC3339),
		"active_connections":   gs.activeConns.Load(),
		"accepted_connections": gs.acceptedConns.Load(),
	}

	if le != "" {
		body["last_error"] = le
	}
	if gs.client != nil {
		body["client"] = gs.client.Stats()
	}
	if gs.tun != nil {
		body["tun"] = gs.tun.Stats()
	}
	if gs.dns != nil {
		body["dns"] = gs.dns.Stats()
	}

	return json.Marshal(body)
}

// --- CLI parsing ---

func parseClientConfig(args []string) (clientConfig, error) {
	var cfg clientConfig
	fs := flag.NewFlagSet("mist-client", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.configPath, "config", "", "path to TOML config file")
	fs.StringVar(&cfg.listen, "l", "127.0.0.1:1080", "socks5/http listen address")
	fs.StringVar(&cfg.serverAddr, "s", "", "server address or mist:// link")
	fs.StringVar(&cfg.sni, "sni", "", "server name indication")
	fs.StringVar(&cfg.password, "p", "", "password")
	fs.StringVar(&cfg.tlsCertSha256, "tls-cert-sha256", "", "expected server certificate SHA-256 fingerprint")
	fs.BoolVar(&cfg.insecure, "insecure", false, "allow insecure TLS connection")
	fs.IntVar(&cfg.minIdleSession, "m", 5, "reserved min idle session")
	fs.StringVar(&cfg.tlsMinVersion, "tls-min-version", "1.2", "minimum TLS version (1.2 or 1.3)")
	fs.StringVar(&cfg.transport, "transport", "tls", "transport mode: tls or wss")
	fs.StringVar(&cfg.tlsProfile, "tls-profile", "", "TLS profile: default or web")
	fs.StringVar(&cfg.trafficProfile, "traffic-profile", "", "padding traffic profile: web, api, or random")
	fs.BoolVar(&cfg.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&cfg.showVersionJSON, "version-json", false, "print version JSON and exit")
	fs.BoolVar(&cfg.check, "check", false, "validate configuration and exit")
	fs.BoolVar(&cfg.check, "check-config", false, "validate configuration and exit")
	fs.StringVar(&cfg.statusJSON, "status-json", "", "write runtime status JSON to this path")
	fs.StringVar(&cfg.logFormat, "log-format", "text", "log format: text or json")
	fs.StringVar(&cfg.logFile, "log-file", "", "append logs to this file while keeping stderr output")
	fs.StringVar(&cfg.inbound, "inbound", "socks,http", "comma-separated inbounds: socks,http,redirect,tun,dns")
	fs.StringVar(&cfg.redirectListen, "redirect-listen", "127.0.0.1:12345", "transparent redirect listen address")

	// TUN flags.
	fs.BoolVar(&cfg.tunEnabled, "tun", false, "enable TUN device mode")
	fs.StringVar(&cfg.tunName, "tun-name", "mist", "TUN interface name")
	fs.IntVar(&cfg.tunMTU, "tun-mtu", 1500, "TUN MTU")
	fs.StringVar(&cfg.tunAddr, "tun-addr", "10.0.0.2/24", "TUN interface address (CIDR)")
	fs.StringVar(&cfg.tunDNS, "tun-dns", "", "comma-separated DNS servers for TUN")
	fs.StringVar(&cfg.tunRoutes, "tun-routes", "", "comma-separated routes through tunnel (default: 0.0.0.0/0)")

	// DNS flags.
	fs.BoolVar(&cfg.dnsEnabled, "dns", false, "enable DNS proxy")
	fs.StringVar(&cfg.dnsListen, "dns-listen", "127.0.0.1:5353", "DNS proxy listen address")
	fs.StringVar(&cfg.dnsUpstream, "dns-upstream", "", "comma-separated upstream DNS servers")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	if serverURL, err := url.Parse(cfg.serverAddr); err == nil && serverURL.Scheme == "mist" {
		cfg.serverAddr = serverURL.Host
		if serverURL.User != nil {
			cfg.password = serverURL.User.String()
		}
		query := serverURL.Query()
		cfg.sni = query.Get("sni")
		if query.Has("insecure") {
			cfg.insecure = parseBoolQuery(query.Get("insecure"))
		}
	}

	return cfg, nil
}

func (cfg clientConfig) clientOptions() (mistclient.Options, error) {
	var tlsMinVersion uint16
	switch cfg.tlsMinVersion {
	case "1.3":
		tlsMinVersion = tls.VersionTLS13
	case "1.2":
		tlsMinVersion = tls.VersionTLS12
	default:
		return mistclient.Options{}, fmt.Errorf("tls-min-version must be 1.2 or 1.3")
	}

	opts := mistclient.Options{
		ServerAddr:     cfg.serverAddr,
		Password:       cfg.password,
		SNI:            cfg.sni,
		TLSCertSHA256:  cfg.tlsCertSha256,
		Insecure:       cfg.insecure,
		TLSMinVersion:  tlsMinVersion,
		MinIdleSession: cfg.minIdleSession,
		Transport:      cfg.transport,
		TLSProfile:     cfg.tlsProfile,
		TrafficProfile: cfg.trafficProfile,
		Logger:         &logrusAdapter{},
	}
	opts.SetDefaults()
	if err := opts.Validate(); err != nil {
		return mistclient.Options{}, err
	}
	return opts, nil
}

// --- Status reporter (file-based JSON, backward compatible) ---

type statusReporter struct {
	path           string
	serverAddr     string
	listen         string
	redirectListen string
	inbound        string
	globalStatus   *globalStatus
	tickerStop     chan struct{}
	activeConns    *atomic.Int64
	acceptedConns  *atomic.Int64
}

func newStatusReporter(path, serverAddr, listen, redirectListen, inbound string, client *mistclient.Client, gs *globalStatus) *statusReporter {
	return &statusReporter{
		path:           strings.TrimSpace(path),
		serverAddr:     serverAddr,
		listen:         listen,
		redirectListen: redirectListen,
		inbound:        inbound,
		globalStatus:   gs,
		tickerStop:     make(chan struct{}),
		activeConns:    &gs.activeConns,
		acceptedConns:  &gs.acceptedConns,
	}
}

func (r *statusReporter) Start() {
	if r.path == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.Write()
			case <-r.tickerStop:
				return
			}
		}
	}()
}

func (r *statusReporter) Stop() {
	close(r.tickerStop)
}

func (r *statusReporter) SetServerState(state string) {
	r.globalStatus.SetServerState(state)
	r.Write()
}

func (r *statusReporter) SetError(err error) {
	if err == nil {
		return
	}
	r.globalStatus.SetError(err)
	r.Write()
}

func (r *statusReporter) Accepted() {
	r.acceptedConns.Add(1)
	r.Write()
}

func (r *statusReporter) ConnectionStarted() {
	r.activeConns.Add(1)
	r.Write()
}

func (r *statusReporter) ConnectionDone() {
	r.activeConns.Add(-1)
	r.Write()
}

func (r *statusReporter) Write() {
	tmp := r.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		logrus.Errorln("write status json:", err)
		return
	}
	data, err := r.globalStatus.StatusJSON()
	if err != nil {
		logrus.Errorln("marshal status json:", err)
		f.Close()
		return
	}
	f.Write(data)
	f.Close()
	if err := os.Rename(tmp, r.path); err != nil {
		logrus.Errorln("rename status json:", err)
	}
}

// --- Logging ---

func configureLogging(format, filePath string) error {
	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		return fmt.Errorf("log-format must be text or json")
	}

	if strings.TrimSpace(filePath) == "" {
		return nil
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	logrus.SetOutput(io.MultiWriter(os.Stderr, f))
	return nil
}

func acceptLoop(ctx context.Context, listener net.Listener, status *statusReporter, handle func(net.Conn)) error {
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			status.SetError(err)
			return fmt.Errorf("accept: %w", err)
		}
		status.Accepted()
		go func() {
			status.ConnectionStarted()
			defer status.ConnectionDone()
			handle(conn)
		}()
	}
}

func printVersionJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(map[string]string{
		"name":    "mist-client",
		"version": util.Version,
		"commit":  util.Commit,
		"date":    util.Date,
	})
}

func parseBoolQuery(value string) bool {
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func splitCSV(s string) []string {
	var parts []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			parts = append(parts, item)
		}
	}
	return parts
}

// logrusAdapter adapts logrus to the mistclient.Logger interface.
type logrusAdapter struct{}

func (logrusAdapter) Info(args ...any)                  { logrus.Infoln(args...) }
func (logrusAdapter) Infof(format string, args ...any)  { logrus.Infof(format, args...) }
func (logrusAdapter) Error(args ...any)                 { logrus.Errorln(args...) }
func (logrusAdapter) Errorf(format string, args ...any) { logrus.Errorf(format, args...) }
func (logrusAdapter) Debug(args ...any)                 { logrus.Debugln(args...) }
func (logrusAdapter) Debugf(format string, args ...any) { logrus.Debugf(format, args...) }
func (logrusAdapter) Warn(args ...any)                  { logrus.Warnln(args...) }
func (logrusAdapter) Warnf(format string, args ...any)  { logrus.Warnf(format, args...) }
func (logrusAdapter) Fatal(args ...any)                 { logrus.Fatalln(args...) }
func (logrusAdapter) Fatalf(format string, args ...any) { logrus.Fatalf(format, args...) }
