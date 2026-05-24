//go:build windows

package proxy

import (
	"net"
	"time"
)

func SetTCPFastOpen(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}
}

var SystemDialer = &net.Dialer{
	Timeout: time.Second * 5,
}
