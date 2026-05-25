package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mist/mistclient"
	"mist/util"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type clientConfig struct {
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
	showVersion     bool
	showVersionJSON bool
	check           bool
	statusJSON      string
	logFormat       string
	logFile         string
}

type inboundSet struct {
	socksHTTP bool
	redirect  bool
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
	inbounds, err := parseInboundSet(cfg.inbound)
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

	status := newStatusReporter(cfg.statusJSON, cfg.serverAddr, cfg.listen, cfg.redirectListen, cfg.inbound, client)
	status.SetServerState("running")
	status.Write()
	status.Start()
	defer status.Stop()

	logrus.Infoln("[Client]", util.ProgramVersionName)
	logrus.Infoln("[Client] inbound", cfg.inbound, "=>", cfg.serverAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 2)
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

func parseClientConfig(args []string) (clientConfig, error) {
	var cfg clientConfig
	fs := flag.NewFlagSet("mist-client", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.listen, "l", "127.0.0.1:1080", "socks5/http listen address")
	fs.StringVar(&cfg.serverAddr, "s", "", "server address or mist:// link")
	fs.StringVar(&cfg.sni, "sni", "", "server name indication")
	fs.StringVar(&cfg.password, "p", "", "password")
	fs.StringVar(&cfg.tlsCertSha256, "tls-cert-sha256", "", "expected server certificate SHA-256 fingerprint")
	fs.BoolVar(&cfg.insecure, "insecure", false, "allow insecure TLS connection")
	fs.IntVar(&cfg.minIdleSession, "m", 5, "reserved min idle session")
	fs.StringVar(&cfg.tlsMinVersion, "tls-min-version", "1.2", "minimum TLS version (1.2 or 1.3)")
	fs.BoolVar(&cfg.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&cfg.showVersionJSON, "version-json", false, "print version JSON and exit")
	fs.BoolVar(&cfg.check, "check", false, "validate configuration and exit")
	fs.BoolVar(&cfg.check, "check-config", false, "validate configuration and exit")
	fs.StringVar(&cfg.statusJSON, "status-json", "", "write runtime status JSON to this path")
	fs.StringVar(&cfg.logFormat, "log-format", "text", "log format: text or json")
	fs.StringVar(&cfg.logFile, "log-file", "", "append logs to this file while keeping stderr output")
	fs.StringVar(&cfg.inbound, "inbound", "socks,http", "comma-separated inbounds: socks,http,redirect")
	fs.StringVar(&cfg.redirectListen, "redirect-listen", "127.0.0.1:12345", "transparent redirect listen address")
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
		Logger:         &logrusAdapter{},
	}
	opts.SetDefaults()
	if err := opts.Validate(); err != nil {
		return mistclient.Options{}, err
	}
	return opts, nil
}

func parseInboundSet(value string) (inboundSet, error) {
	var set inboundSet
	for _, item := range strings.Split(value, ",") {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "":
		case "socks", "http":
			set.socksHTTP = true
		case "redirect":
			set.redirect = true
		default:
			return set, fmt.Errorf("unsupported inbound %q", item)
		}
	}
	if !set.socksHTTP && !set.redirect {
		return set, fmt.Errorf("at least one inbound is required")
	}
	return set, nil
}

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

type statusReporter struct {
	path           string
	serverAddr     string
	listen         string
	redirectListen string
	inbound        string
	client         *mistclient.Client
	startedAt      time.Time
	tickerStop     chan struct{}
	activeConns    atomic.Int64
	acceptedConns  atomic.Int64

	mu          sync.Mutex
	serverState string
	lastError   string
}

func newStatusReporter(path, serverAddr, listen, redirectListen, inbound string, client *mistclient.Client) *statusReporter {
	return &statusReporter{
		path:           strings.TrimSpace(path),
		serverAddr:     serverAddr,
		listen:         listen,
		redirectListen: redirectListen,
		inbound:        inbound,
		client:         client,
		startedAt:      time.Now().UTC(),
		tickerStop:     make(chan struct{}),
		serverState:    "starting",
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
	r.mu.Lock()
	r.serverState = state
	r.mu.Unlock()
	r.Write()
}

func (r *statusReporter) SetError(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	r.lastError = err.Error()
	r.mu.Unlock()
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
	if r.path == "" {
		return
	}

	r.mu.Lock()
	serverState := r.serverState
	lastError := r.lastError
	r.mu.Unlock()

	body := struct {
		Version        string           `json:"version"`
		Commit         string           `json:"commit"`
		Date           string           `json:"date"`
		Server         string           `json:"server"`
		ServerState    string           `json:"server_state"`
		Inbound        string           `json:"inbound"`
		Listen         string           `json:"listen"`
		RedirectListen string           `json:"redirect_listen,omitempty"`
		StartedAt      string           `json:"started_at"`
		UpdatedAt      string           `json:"updated_at"`
		ActiveConns    int64            `json:"active_connections"`
		AcceptedConns  int64            `json:"accepted_connections"`
		LastError      string           `json:"last_error,omitempty"`
		Client         mistclient.Stats `json:"client"`
	}{
		Version:        util.Version,
		Commit:         util.Commit,
		Date:           util.Date,
		Server:         r.serverAddr,
		ServerState:    serverState,
		Inbound:        r.inbound,
		Listen:         r.listen,
		RedirectListen: r.redirectListen,
		StartedAt:      r.startedAt.Format(time.RFC3339),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
		ActiveConns:    r.activeConns.Load(),
		AcceptedConns:  r.acceptedConns.Load(),
		LastError:      lastError,
		Client:         r.client.Stats(),
	}

	tmp := r.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		logrus.Errorln("write status json:", err)
		return
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(body); err != nil {
		logrus.Errorln("encode status json:", err)
	}
	if err := f.Close(); err != nil {
		logrus.Errorln("close status json:", err)
		return
	}
	if err := os.Rename(tmp, r.path); err != nil {
		logrus.Errorln("rename status json:", err)
	}
}

func parseBoolQuery(value string) bool {
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
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
