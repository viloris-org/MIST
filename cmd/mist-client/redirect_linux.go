//go:build linux

package main

import (
	"context"
	"fmt"
	"mist/mistclient"
	"net"
	"net/netip"
	"unsafe"

	M "MistCore/common/metadata"
	"golang.org/x/sys/unix"
)

func handleRedirectConnection(ctx context.Context, conn net.Conn, client *mistclient.Client) {
	defer conn.Close()

	dst, err := originalDst(conn)
	if err != nil {
		logrusAdapter{}.Errorf("redirect original dst: %v", err)
		return
	}

	metadata := M.Metadata{
		Source:      M.SocksaddrFromNet(conn.RemoteAddr()),
		Destination: dst,
	}
	if err := client.NewConnection(ctx, conn, metadata); err != nil {
		logrusAdapter{}.Errorf("redirect connection: %v", err)
	}
}

func originalDst(conn net.Conn) (M.Socksaddr, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return M.Socksaddr{}, fmt.Errorf("connection is %T, not *net.TCPConn", conn)
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return M.Socksaddr{}, err
	}

	var dst M.Socksaddr
	var sockErr error
	controlErr := rawConn.Control(func(fd uintptr) {
		dst, sockErr = originalDstFD(int(fd))
	})
	if controlErr != nil {
		return M.Socksaddr{}, controlErr
	}
	if sockErr != nil {
		return M.Socksaddr{}, sockErr
	}
	return dst, nil
}

func originalDstFD(fd int) (M.Socksaddr, error) {
	if dst, err := originalDst4(fd); err == nil {
		return dst, nil
	}
	return originalDst6(fd)
}

func originalDst4(fd int) (M.Socksaddr, error) {
	var raw unix.RawSockaddrInet4
	size := uint32(unsafe.Sizeof(raw))
	if err := getsockoptRaw(fd, unix.SOL_IP, unix.SO_ORIGINAL_DST, unsafe.Pointer(&raw), &size); err != nil {
		return M.Socksaddr{}, err
	}
	addr := netip.AddrFrom4(raw.Addr)
	return M.SocksaddrFrom(addr, sockaddrPort(raw.Port)), nil
}

func originalDst6(fd int) (M.Socksaddr, error) {
	var raw unix.RawSockaddrInet6
	size := uint32(unsafe.Sizeof(raw))
	if err := getsockoptRaw(fd, unix.SOL_IPV6, unix.SO_ORIGINAL_DST, unsafe.Pointer(&raw), &size); err != nil {
		return M.Socksaddr{}, err
	}
	addr := netip.AddrFrom16(raw.Addr)
	return M.SocksaddrFrom(addr, sockaddrPort(raw.Port)), nil
}

func sockaddrPort(port uint16) uint16 {
	bytes := (*[2]byte)(unsafe.Pointer(&port))
	return uint16(bytes[0])<<8 | uint16(bytes[1])
}

func getsockoptRaw(fd, level, opt int, value unsafe.Pointer, vallen *uint32) error {
	_, _, errno := unix.Syscall6(
		unix.SYS_GETSOCKOPT,
		uintptr(fd),
		uintptr(level),
		uintptr(opt),
		uintptr(value),
		uintptr(unsafe.Pointer(vallen)),
		0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
