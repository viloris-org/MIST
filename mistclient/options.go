package mistclient

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Options struct {
	ServerAddr string
	Password   string

	// TLS
	SNI            string // override ServerName in TLS ClientHello
	TLSCertSHA256  string // hex-encoded SHA-256 of server cert DER for pinning
	Insecure       bool   // skip all TLS verification
	TLSMinVersion  uint16 // 0 means tls.VersionTLS12
	KeyLogWriter   io.Writer // optional TLS key log writer

	// Session pool
	MinIdleSession     int
	MaxStreams         int
	ReadTimeout        time.Duration
	KeepaliveInterval  time.Duration
	SynRateLimit       int

	// Logger, defaults to silent if nil
	Logger Logger
}

// SetDefaults fills in sane defaults for zero-value fields.
func (o *Options) SetDefaults() {
	if o.TLSMinVersion == 0 {
		o.TLSMinVersion = tls.VersionTLS12
	}
	if o.MinIdleSession == 0 {
		o.MinIdleSession = 1
	}
	if o.Logger == nil {
		o.Logger = nopLogger{}
	}
}

// Validate returns an error if required fields are missing or invalid.
func (o *Options) Validate() error {
	if o.ServerAddr == "" {
		return fmt.Errorf("ServerAddr is required")
	}
	if o.Password == "" {
		return fmt.Errorf("Password is required")
	}
	return nil
}

// PasswordHash returns the SHA-256 hash of the password.
func (o *Options) PasswordHash() []byte {
	sum := sha256.Sum256([]byte(o.Password))
	return sum[:]
}

// TLSConfig builds a *tls.Config from the options.
func (o *Options) TLSConfig() *tls.Config {
	serverHost, _, err := net.SplitHostPort(o.ServerAddr)
	if err != nil {
		serverHost = o.ServerAddr
	}

	sni := strings.TrimSpace(o.SNI)
	if sni == "" && net.ParseIP(serverHost) == nil {
		sni = serverHost
	}

	cfg := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: o.Insecure || o.TLSCertSHA256 != "",
		MinVersion:         o.TLSMinVersion,
		ClientSessionCache: tls.NewLRUClientSessionCache(64),
		KeyLogWriter:       o.KeyLogWriter,
	}

	if o.TLSCertSHA256 != "" {
		pin := parseCertPin(o.TLSCertSHA256)
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
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

	return cfg
}

func parseCertPin(pin string) []byte {
	pin = strings.TrimSpace(strings.ToLower(pin))
	pin = strings.TrimPrefix(pin, "sha256:")
	pin = strings.ReplaceAll(pin, ":", "")
	decoded, _ := hex.DecodeString(pin)
	return decoded
}
