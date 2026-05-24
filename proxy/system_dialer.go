package proxy

import (
	"net"
	"syscall"
	"time"
)

func SetTCPFastOpen(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)

		rawConn, err := tcpConn.SyscallConn()
		if err != nil {
			return
		}
		rawConn.Control(func(fd uintptr) {
			// Increase buffer sizes for better throughput over high-BDP links
			syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 256*1024)
			syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 256*1024)
		})
	}
}

var SystemDialer = &net.Dialer{
	Timeout: time.Second * 5,
	Control: func(network, address string, c syscall.RawConn) error {
		var err error
		c.Control(func(fd uintptr) {
			err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 256*1024)
		})
		if err != nil {
			return err
		}
		c.Control(func(fd uintptr) {
			err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 256*1024)
		})
		if err != nil {
			return err
		}
		return nil
	},
}
