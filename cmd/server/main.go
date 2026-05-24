package main

import (
	"mist/proxy/padding"
	"mist/util"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
)

var passwordSha256 []byte

func main() {
	listen := flag.String("l", "0.0.0.0:8443", "server listen port")
	password := flag.String("p", "", "password")
	paddingScheme := flag.String("padding-scheme", "", "padding-scheme")
	certType := flag.String("cert-type", "self-signed", "certificate type: self-signed (or self/ip), acme (or domain), or custom")
	certName := flag.String("cert-name", "", "certificate IP address or domain name")
	acmeHTTP := flag.String("acme-http", ":80", "ACME HTTP-01 challenge listen address")
	acmeCache := flag.String("acme-cache", "cert-cache", "ACME certificate cache directory")
	acmeEmail := flag.String("acme-email", "", "ACME account email")
	certFile := flag.String("cert-file", "", "custom certificate file path (required for custom type)")
	keyFile := flag.String("key-file", "", "custom private key file path (required for custom type)")
	fallback := flag.String("fallback", "", "fallback address for unauthorized traffic (e.g. 127.0.0.1:80)")
	flag.Parse()

	if *password == "" {
		logrus.Fatalln("please set password")
	}
	if *paddingScheme != "" {
		if f, err := os.Open(*paddingScheme); err == nil {
			b, err := io.ReadAll(f)
			if err != nil {
				logrus.Fatalln(err)
			}
			if padding.UpdatePaddingScheme(b) {
				logrus.Infoln("loaded padding scheme file:", *paddingScheme)
			} else {
				logrus.Errorln("wrong format padding scheme file:", *paddingScheme)
			}
			f.Close()
		} else {
			logrus.Fatalln(err)
		}
	}

	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	var sum = sha256.Sum256([]byte(*password))
	passwordSha256 = sum[:]

	logrus.Infoln("[Server]", util.ProgramVersionName)
	logrus.Infoln("[Server] Listening TCP", *listen)
	if *fallback != "" {
		logrus.Infoln("[Server] Fallback address:", *fallback)
	}

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		logrus.Fatalln("listen server tcp:", err)
	}

	tlsConfig, err := newServerTLSConfig(*certType, *certName, *listen, *acmeHTTP, *acmeCache, *acmeEmail, *certFile, *keyFile)
	if err != nil {
		logrus.Fatalln("error certificate options:", err)
	}

	ctx := context.Background()
	server := NewMyServer(tlsConfig, *fallback)

	for {
		c, err := listener.Accept()
		if err != nil {
			logrus.Fatalln("accept:", err)
		}
		go handleTcpConnection(ctx, c, server)
	}
}

func newServerTLSConfig(certType, certName, listen, acmeHTTP, acmeCache, acmeEmail, certFile, keyFile string) (*tls.Config, error) {
	certType = strings.ToLower(strings.TrimSpace(certType))
	switch certType {
	case "self-signed", "self":
		return newSelfSignedTLSConfig(certName, listen, false)
	case "ip":
		// Compatibility with older "ip" option which strictly requires an IP
		return newSelfSignedTLSConfig(certName, listen, true)
	case "acme", "domain":
		return newACMETLSConfig(certName, acmeHTTP, acmeCache, acmeEmail)
	case "custom":
		return newCustomTLSConfig(certFile, keyFile)
	default:
		return nil, fmt.Errorf("-cert-type must be self-signed, acme, or custom")
	}
}

func newSelfSignedTLSConfig(certName, listen string, requireIP bool) (*tls.Config, error) {
	generatedCertName, err := generatedSelfSignedCertificateName(certName, listen, requireIP)
	if err != nil {
		return nil, err
	}
	tlsCert, err := util.GenerateKeyPair(time.Now, generatedCertName)
	if err != nil {
		return nil, fmt.Errorf("generate tls certificate: %w", err)
	}
	if len(tlsCert.Certificate) > 0 {
		certSum := sha256.Sum256(tlsCert.Certificate[0])
		logrus.Infof("[Server] TLS self-signed cert %s sha256 %x", generatedCertName, certSum)
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return tlsCert, nil
		},
	}, nil
}

func newCustomTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	certFile = strings.TrimSpace(certFile)
	keyFile = strings.TrimSpace(keyFile)
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("-cert-file and -key-file are required for -cert-type custom")
	}
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load custom certificate key pair: %w", err)
	}
	if len(tlsCert.Certificate) > 0 {
		certSum := sha256.Sum256(tlsCert.Certificate[0])
		logrus.Infof("[Server] TLS custom cert sha256 %x", certSum)
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &tlsCert, nil
		},
	}, nil
}

func newACMETLSConfig(certName, acmeHTTP, acmeCache, acmeEmail string) (*tls.Config, error) {
	domain, err := acmeDomainName(certName)
	if err != nil {
		return nil, err
	}
	acmeHTTP = strings.TrimSpace(acmeHTTP)
	if acmeHTTP == "" {
		return nil, fmt.Errorf("-acme-http is required for -cert-type domain")
	}
	acmeCache = strings.TrimSpace(acmeCache)
	if acmeCache == "" {
		return nil, fmt.Errorf("-acme-cache is required for -cert-type domain")
	}

	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(acmeCache),
		HostPolicy: autocert.HostWhitelist(domain),
		Email:      strings.TrimSpace(acmeEmail),
	}
	challengeListener, err := net.Listen("tcp", acmeHTTP)
	if err != nil {
		return nil, fmt.Errorf("listen ACME HTTP-01 challenge server: %w", err)
	}
	go func() {
		if err := http.Serve(challengeListener, manager.HTTPHandler(nil)); err != nil && err != http.ErrServerClosed {
			logrus.Fatalln("serve ACME HTTP-01 challenge:", err)
		}
	}()

	logrus.Infof("[Server] TLS ACME domain cert %s cache %s http-01 %s", domain, acmeCache, acmeHTTP)
	tlsConfig := manager.TLSConfig()
	tlsConfig.MinVersion = tls.VersionTLS12
	return tlsConfig, nil
}

func generatedIPCertificateName(certName, listen string) (string, error) {
	return generatedSelfSignedCertificateName(certName, listen, true)
}

func generatedSelfSignedCertificateName(certName, listen string, requireIP bool) (string, error) {
	certName = strings.TrimSpace(certName)

	if certName == "" {
		host, _, err := net.SplitHostPort(listen)
		if err != nil {
			return "", err
		}
		ip := net.ParseIP(host)
		if ip == nil || ip.IsUnspecified() {
			certName = "127.0.0.1"
		} else {
			certName = ip.String()
		}
	}
	if requireIP && net.ParseIP(certName) == nil {
		return "", fmt.Errorf("-cert-type ip requires -cert-name to be an IP address")
	}
	if strings.ContainsAny(certName, "/:") {
		return "", fmt.Errorf("invalid self-signed certificate name: %s", certName)
	}
	return certName, nil
}

func acmeDomainName(certName string) (string, error) {
	certName = strings.ToLower(strings.TrimSpace(certName))
	if certName == "" {
		return "", fmt.Errorf("-cert-type domain requires -cert-name")
	}
	if net.ParseIP(certName) != nil {
		return "", fmt.Errorf("-cert-type domain requires -cert-name to be a domain name")
	}
	if strings.ContainsAny(certName, "/:") {
		return "", fmt.Errorf("-cert-name must be a bare domain name")
	}
	return certName, nil
}
