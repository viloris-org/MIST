package main

import (
	std_bufio "bufio"
	"context"
	"net"
	"runtime/debug"

	"mist/mistclient"

	M "MistCore/common/metadata"
	"MistCore/protocol/http"
	"MistCore/protocol/socks"
	"MistCore/protocol/socks/socks4"
	"MistCore/protocol/socks/socks5"
	"github.com/sirupsen/logrus"
)

func handleTcpConnection(ctx context.Context, c net.Conn, s *mistclient.Client) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorln("[BUG]", r, string(debug.Stack()))
		}
	}()
	defer c.Close()

	reader := std_bufio.NewReader(c)
	headerBytes, err := reader.Peek(1)
	if err != nil {
		return
	}

	metadata := M.Metadata{
		Source:      M.SocksaddrFromNet(c.RemoteAddr()),
		Destination: M.SocksaddrFromNet(c.LocalAddr()),
	}

	switch headerBytes[0] {
	case socks4.Version, socks5.Version:
		socks.HandleConnection0(ctx, c, reader, nil, s, metadata)
	default:
		http.HandleConnection(ctx, c, reader, nil, s, metadata)
	}
}
