//go:build !windows

package proxy

import (
	"net"
	"runtime"
	"syscall"
	"time"
)

const (
	tcpBufferSize      = 1024 * 1024
	tcpNotSentLowWater = 128 * 1024
)

func SetTCPFastOpen(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)

		rawConn, err := tcpConn.SyscallConn()
		if err != nil {
			return
		}
		rawConn.Control(func(fd uintptr) {
			setTCPOptions(fd)
		})
	}
}

var SystemDialer = &net.Dialer{
	Timeout: time.Second * 5,
	Control: func(network, address string, c syscall.RawConn) error {
		var err error
		c.Control(func(fd uintptr) {
			err = setTCPOptions(fd)
		})
		if err != nil {
			return err
		}
		return nil
	},
}

func setTCPOptions(fd uintptr) error {
	if err := syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
		return err
	}
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, tcpBufferSize); err != nil {
		return err
	}
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, tcpBufferSize); err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		// Keep kernel-side unsent queues bounded so weak links do not build
		// seconds of stale data before newer stream/control frames can move.
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, 25, tcpNotSentLowWater)
	}
	return nil
}
