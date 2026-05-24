package main

import (
	"mist/mistclient"
	"mist/util"
	"context"
	"flag"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func main() {
	listen := flag.String("l", "127.0.0.1:1080", "socks5 listen port")
	serverAddr := flag.String("s", "", "Server address or mist:// link")
	sni := flag.String("sni", "", "Server Name Indication")
	password := flag.String("p", "", "Password")
	tlsCertSha256 := flag.String("tls-cert-sha256", "", "expected server certificate SHA-256 fingerprint")
	insecure := flag.Bool("insecure", false, "allow insecure TLS connection")
	minIdleSession := flag.Int("m", 5, "Reserved min idle session")
	tlsMinVersionStr := flag.String("tls-min-version", "1.2", "minimum TLS version (1.2 or 1.3)")
	flag.Parse()

	if serverURL, err := url.Parse(*serverAddr); err == nil {
		if serverURL.Scheme == "mist" {
			*serverAddr = serverURL.Host
			if serverURL.User != nil {
				*password = serverURL.User.String()
			}
			query := serverURL.Query()
			*sni = query.Get("sni")
			if query.Has("insecure") {
				*insecure = parseBoolQuery(query.Get("insecure"))
			}
		}
	}

	if *serverAddr == "" {
		logrus.Fatalln("please set -s server adreess")
	}

	if *password == "" {
		logrus.Fatalln("please set -p password")
	}

	var tlsMinVersion uint16
	switch *tlsMinVersionStr {
	case "1.3":
		tlsMinVersion = 0x0304 // tls.VersionTLS13
	case "1.2":
		tlsMinVersion = 0x0303 // tls.VersionTLS12
	default:
		logrus.Fatalln("tls-min-version must be 1.2 or 1.3")
	}

	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	logrus.Infoln("[Client]", util.ProgramVersionName)
	logrus.Infoln("[Client] socks5/http", *listen, "=>", *serverAddr)

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		logrus.Fatalln("listen socks5 tcp:", err)
	}

	opts := mistclient.Options{
		ServerAddr:     *serverAddr,
		Password:       *password,
		SNI:            *sni,
		TLSCertSHA256:  *tlsCertSha256,
		Insecure:       *insecure,
		TLSMinVersion:  tlsMinVersion,
		MinIdleSession: *minIdleSession,
		Logger:         &logrusAdapter{},
	}
	if path := strings.TrimSpace(os.Getenv("TLS_KEY_LOG")); path != "" {
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644); err == nil {
			opts.KeyLogWriter = f
		}
	}

	client, err := mistclient.NewClient(opts)
	if err != nil {
		logrus.Fatalln("create client:", err)
	}

	ctx := context.Background()
	for {
		c, err := listener.Accept()
		if err != nil {
			logrus.Fatalln("accept:", err)
		}
		go handleTcpConnection(ctx, c, client)
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
func (logrusAdapter) Errorf(format string, args ...any)  { logrus.Errorf(format, args...) }
func (logrusAdapter) Debug(args ...any)                 { logrus.Debugln(args...) }
func (logrusAdapter) Debugf(format string, args ...any)  { logrus.Debugf(format, args...) }
func (logrusAdapter) Warn(args ...any)                  { logrus.Warnln(args...) }
func (logrusAdapter) Warnf(format string, args ...any)   { logrus.Warnf(format, args...) }
func (logrusAdapter) Fatal(args ...any)                 { logrus.Fatalln(args...) }
func (logrusAdapter) Fatalf(format string, args ...any)  { logrus.Fatalf(format, args...) }
