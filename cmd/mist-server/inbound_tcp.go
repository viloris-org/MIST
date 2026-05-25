package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/binary"
	"mist/proxy"
	"mist/proxy/padding"
	"mist/proxy/session"
	"net"
	"runtime/debug"
	"strings"
	"time"

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

	by, err := b.ReadBytes(32)
	if err != nil || subtle.ConstantTimeCompare(by, passwordSha256) != 1 {
		_ = c.SetReadDeadline(time.Time{})
		b.Resize(0, n)
		fallback(ctx, c, s.fallbackAddr)
		return
	}
	by, err = b.ReadBytes(2)
	if err != nil {
		_ = c.SetReadDeadline(time.Time{})
		b.Resize(0, n)
		fallback(ctx, c, s.fallbackAddr)
		return
	}
	paddingLen := binary.BigEndian.Uint16(by)
	if paddingLen > 0 {
		_, err = b.ReadBytes(int(paddingLen))
		if err != nil {
			_ = c.SetReadDeadline(time.Time{})
			b.Resize(0, n)
			fallback(ctx, c, s.fallbackAddr)
			return
		}
	}
	_ = c.SetReadDeadline(time.Time{})

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
	}, &padding.DefaultPaddingFactory, s.maxStreams, s.readTimeout, s.keepaliveInterval, s.synRateLimit, s.passwordHash)
	session.Run()
	session.Close()
}

func fallback(ctx context.Context, c net.Conn, fallbackAddr string) {
	fallbackAddr = strings.TrimSpace(fallbackAddr)
	if fallbackAddr == "" {
		// 返回标准的 400 Bad Request
		_, _ = c.Write([]byte("HTTP/1.1 400 Bad Request\r\nConnection: close\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n400 Bad Request\n"))
		logrus.Debugln("fallback: no fallback address configured, returned HTTP 400 to", c.RemoteAddr())
		return
	}

	// 清洗和规范化 fallback 地址
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
