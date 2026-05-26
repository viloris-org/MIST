package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"mist/proxy"
	"mist/proxy/padding"
	"mist/proxy/session"
	"net"
	"runtime/debug"
	"strings"
	"time"

	"MistCore/common/atomic"
	"MistCore/common/buf"
	"MistCore/common/bufio"
	M "MistCore/common/metadata"
	"MistCore/common/uot"
	"github.com/sirupsen/logrus"
)

func handleTcpConnection(ctx context.Context, c net.Conn, s *myServer) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorln("[BUG]", r, string(debug.Stack()))
		}
	}()

	proxy.SetTCPFastOpen(c)
	c = tls.Server(c, s.tlsConfig)
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))

	b := buf.NewPacket()
	defer b.Release()

	n, err := b.ReadOnceFrom(c)
	if err != nil {
		logrus.Debugln("ReadOnceFrom:", err)
		return
	}
	c = bufio.NewCachedConn(c, b)

	// Phase 1: try legacy raw-hash format.
	authenticated := false
	if tryLegacyAuth(b, passwordSha256) {
		authenticated = true
	}

	// Phase 2: try HTTP-embedded auth format.
	if !authenticated {
		b.Resize(0, n)
		if tryHTTPAuth(b, passwordSha256) {
			// Discard remaining body bytes (padding).
			b.Resize(0, 0)
			authenticated = true
		}
	}

	if !authenticated {
		_ = c.SetReadDeadline(time.Time{})
		b.Resize(0, n)
		fallback(ctx, c, s.fallbackAddr)
		return
	}
	_ = c.SetReadDeadline(time.Time{})

	s.SessionAccepted()
	defer s.SessionClosed()

	// Generate a fresh random padding scheme per session.
	var sessionPadding atomic.TypedValue[*padding.PaddingFactory]
	if serverPaddingSchemeExplicit {
		sessionPadding.Store(padding.DefaultPaddingFactory.Load())
	} else {
		randomScheme, err := padding.GenerateRandomScheme()
		if err == nil {
			if f := padding.NewPaddingFactory(randomScheme); f != nil {
				sessionPadding.Store(f)
			}
		}
		if sessionPadding.Load() == nil {
			sessionPadding.Store(padding.DefaultPaddingFactory.Load())
		}
	}

	session := session.NewServerSession(c, func(stream *session.Stream) {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorln("[BUG]", r, string(debug.Stack()))
			}
		}()
		defer stream.Close()

		destination, err := M.SocksaddrSerializer.ReadAddrPort(stream)
		if err != nil {
			logrus.Debugln("ReadAddrPort:", err)
			return
		}

		if destination.Fqdn == uot.MagicAddress || destination.Fqdn == uot.LegacyMagicAddress {
			proxyOutboundUoT(ctx, stream, destination)
		} else {
			proxyOutboundTCP(ctx, stream, destination)
		}
	}, &sessionPadding, s.maxStreams, s.streamBufferSize, s.readTimeout, s.keepaliveInterval, s.synRateLimit, s.passwordHash)
	session.Run()
	session.Close()
}

func tryLegacyAuth(b *buf.Buffer, passwordHash []byte) bool {
	rawData := b.Bytes()
	if len(rawData) < 34 {
		return false
	}
	hash := rawData[:32]
	if subtle.ConstantTimeCompare(hash, passwordHash) != 1 {
		return false
	}
	b.Advance(32)
	if len(b.Bytes()) < 2 {
		return false
	}
	padLen := int(b.Bytes()[0])<<8 | int(b.Bytes()[1])
	b.Advance(2)
	if len(b.Bytes()) < padLen {
		return false
	}
	if padLen > 0 {
		b.Advance(padLen)
	}
	return true
}

// tryHTTPAuth attempts to parse a fake HTTP request and extract the password
// hash from the Authorization: Bearer header. Returns true on success.
func tryHTTPAuth(b *buf.Buffer, passwordHash []byte) bool {
	data := b.Bytes()
	// Find end of HTTP headers.
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd < 0 {
		return false
	}
	headers := data[:headerEnd]

	// Find Authorization: Bearer <token>
	token := extractBearerToken(headers)
	if len(token) == 0 {
		return false
	}

	// Decode base64.
	decoded := make([]byte, base64.RawURLEncoding.DecodedLen(len(token)))
	n, err := base64.RawURLEncoding.Decode(decoded, token)
	if err != nil {
		n, err = base64.StdEncoding.Decode(decoded, token)
		if err != nil {
			return false
		}
	}
	decoded = decoded[:n]

	return subtle.ConstantTimeCompare(decoded, passwordHash) == 1
}

func extractBearerToken(headers []byte) []byte {
	for _, prefix := range [][]byte{
		[]byte("Authorization: Bearer "),
		[]byte("authorization: bearer "),
		[]byte("Authorization: bearer "),
		[]byte("AUTHORIZATION: BEARER "),
	} {
		idx := bytes.Index(headers, prefix)
		if idx < 0 {
			continue
		}
		start := idx + len(prefix)
		end := bytes.IndexByte(headers[start:], '\r')
		if end < 0 {
			end = len(headers) - start
		}
		return headers[start : start+end]
	}
	return nil
}

var fallbackHTML = []byte("HTTP/1.1 200 OK\r\n" +
	"Connection: close\r\n" +
	"Content-Type: text/html; charset=utf-8\r\n" +
	"\r\n" +
	"<!DOCTYPE html>\n" +
	"<html lang=\"en\">\n" +
	"<head>\n" +
	"<meta charset=\"utf-8\">\n" +
	"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n" +
	"<title>Service Unavailable</title>\n" +
	"<style>\n" +
	"body { font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; color: #333; }\n" +
	".box { text-align: center; padding: 3rem; background: #fff; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); max-width: 480px; }\n" +
	"h1 { font-size: 1.25rem; font-weight: 500; margin: 0 0 0.75rem; }\n" +
	"p { font-size: 0.875rem; color: #666; margin: 0; line-height: 1.5; }\n" +
	".footer { margin-top: 1.5rem; font-size: 0.75rem; color: #999; }\n" +
	"</style>\n" +
	"</head>\n" +
	"<body>\n" +
	"<div class=\"box\">\n" +
	"<h1>This service is temporarily unavailable.</h1>\n" +
	"<p>Please try again later. If the problem persists, contact the system administrator.</p>\n" +
	"<div class=\"footer\">nginx/1.26.0</div>\n" +
	"</div>\n" +
	"</body>\n" +
	"</html>\n")

func fallback(ctx context.Context, c net.Conn, fallbackAddr string) {
	fallbackAddr = strings.TrimSpace(fallbackAddr)
	if fallbackAddr == "" {
		_, _ = c.Write(fallbackHTML)
		logrus.Debugln("fallback: no fallback address configured, returned HTML to", c.RemoteAddr())
		return
	}

	// Clean and normalize fallback address.
	if strings.HasPrefix(fallbackAddr, "http://") {
		fallbackAddr = strings.TrimPrefix(fallbackAddr, "http://")
	} else if strings.HasPrefix(fallbackAddr, "https://") {
		fallbackAddr = strings.TrimPrefix(fallbackAddr, "https://")
	}
	if idx := strings.Index(fallbackAddr, "/"); idx != -1 {
		fallbackAddr = fallbackAddr[:idx]
	}
	if !strings.Contains(fallbackAddr, ":") {
		fallbackAddr = fallbackAddr + ":80"
	}

	logrus.Infof("fallback: proxying unauthorized connection from %s to %s", c.RemoteAddr(), fallbackAddr)
	backend, err := net.DialTimeout("tcp", fallbackAddr, 5*time.Second)
	if err != nil {
		logrus.Errorln("fallback: failed to dial backend:", err)
		_, _ = c.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n502 Bad Gateway\n"))
		return
	}
	defer backend.Close()

	err = bufio.CopyConn(ctx, c, backend)
	if err != nil {
		logrus.Debugln("fallback copy connection finished:", err)
	}
}
