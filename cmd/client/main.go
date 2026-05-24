package main

import (
	"mist/proxy"
	"mist/util"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

var passwordSha256 []byte

func main() {
	listen := flag.String("l", "127.0.0.1:1080", "socks5 listen port")
	serverAddr := flag.String("s", "", "Server address or mist:// link")
	sni := flag.String("sni", "", "Server Name Indication")
	password := flag.String("p", "", "Password")
	tlsCertSha256 := flag.String("tls-cert-sha256", "", "expected server certificate SHA-256 fingerprint")
	insecure := flag.Bool("insecure", false, "allow insecure TLS connection")
	minIdleSession := flag.Int("m", 5, "Reserved min idle session")
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

	serverHost, _, err := net.SplitHostPort(*serverAddr)
	if err != nil {
		logrus.Fatalln("error server address:", *serverAddr, err)
	}

	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	var sum = sha256.Sum256([]byte(*password))
	passwordSha256 = sum[:]

	logrus.Infoln("[Client]", util.ProgramVersionName)
	logrus.Infoln("[Client] socks5/http", *listen, "=>", *serverAddr)

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		logrus.Fatalln("listen socks5 tcp:", err)
	}

	tlsConfig := &tls.Config{
		ServerName:         serverNameForTLS(serverHost, *sni),
		InsecureSkipVerify: *insecure || *tlsCertSha256 != "",
		MinVersion:         tls.VersionTLS12,
	}
	if *tlsCertSha256 != "" {
		pin, err := parseCertificatePin(*tlsCertSha256)
		if err != nil {
			logrus.Fatalln("error tls-cert-sha256:", err)
		}
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("server did not provide a certificate")
			}
			sum := sha256.Sum256(rawCerts[0])
			if subtle.ConstantTimeCompare(sum[:], pin) != 1 {
				return fmt.Errorf("server certificate pin mismatch")
			}
			return nil
		}
	}

	path := strings.TrimSpace(os.Getenv("TLS_KEY_LOG"))
	if path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err == nil {
			tlsConfig.KeyLogWriter = f
		}
	}

	ctx := context.Background()
	client := NewMyClient(ctx, func(ctx context.Context) (net.Conn, error) {
		conn, err := proxy.SystemDialer.DialContext(ctx, "tcp", *serverAddr)
		if err != nil {
			return nil, err
		}
		conn = tls.Client(conn, tlsConfig)
		return conn, nil
	}, *minIdleSession)

	for {
		c, err := listener.Accept()
		if err != nil {
			logrus.Fatalln("accept:", err)
		}
		go handleTcpConnection(ctx, c, client)
	}
}

func serverNameForTLS(serverHost, sni string) string {
	sni = strings.TrimSpace(sni)
	if sni != "" {
		return sni
	}
	if net.ParseIP(serverHost) == nil {
		return serverHost
	}
	// Go does not send SNI for IP literals.
	return "127.0.0.1"
}

func parseBoolQuery(value string) bool {
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func parseCertificatePin(pin string) ([]byte, error) {
	pin = strings.TrimSpace(strings.ToLower(pin))
	pin = strings.TrimPrefix(pin, "sha256:")
	pin = strings.ReplaceAll(pin, ":", "")
	decoded, err := hex.DecodeString(pin)
	if err != nil {
		return nil, err
	}
	if len(decoded) != sha256.Size {
		return nil, fmt.Errorf("want %d bytes, got %d", sha256.Size, len(decoded))
	}
	return decoded, nil
}
