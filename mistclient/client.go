package mistclient

import (
	"mist/proxy"
	"mist/proxy/padding"
	"mist/proxy/session"
	"mist/util"
	"context"
	"crypto/tls"
	"encoding/binary"
	"net"
	"time"

	"MistCore/common/buf"
	"MistCore/common/bufio"
	M "MistCore/common/metadata"
	"MistCore/common/network"
	"MistCore/common/uot"
)

// Client is a MIST client that manages a pool of multiplexed TLS sessions
// to a MIST server. It is safe for concurrent use from multiple goroutines.
type Client struct {
	opts          Options
	passwordHash  []byte
	tlsConfig     *tls.Config
	sessionClient *session.Client
	die           context.Context
	dieCancel     context.CancelFunc
}

// NewClient creates a new MIST client. It validates the options, fills
// defaults, builds a TLS config, and starts the session pool.
func NewClient(opts Options) (*Client, error) {
	opts.SetDefaults()
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		opts:         opts,
		passwordHash: opts.PasswordHash(),
		tlsConfig:    opts.TLSConfig(),
		die:          ctx,
		dieCancel:    cancel,
	}

	sessionTimeout := time.Second * 30
	c.sessionClient = session.NewClient(
		ctx,
		c.dialSession,
		&padding.DefaultPaddingFactory,
		time.Second*30, // idle check interval
		sessionTimeout,
		opts.MinIdleSession,
		opts.MaxStreams,
		opts.ReadTimeout,
		opts.KeepaliveInterval,
		opts.SynRateLimit,
		c.passwordHash,
	)

	return c, nil
}

// DialStream opens a new stream to the server and writes the destination
// SOCKS address. The returned net.Conn can be used to proxy data to dest.
func (c *Client) DialStream(ctx context.Context, dest M.Socksaddr) (net.Conn, error) {
	conn, err := c.sessionClient.CreateStream(ctx)
	if err != nil {
		return nil, err
	}
	if err := M.SocksaddrSerializer.WriteAddrPort(conn, dest); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// Close shuts down the client and all its sessions.
func (c *Client) Close() error {
	c.dieCancel()
	return c.sessionClient.Close()
}

func (c *Client) dialSession(ctx context.Context) (net.Conn, error) {
	conn, err := proxy.SystemDialer.DialContext(ctx, "tcp", c.opts.ServerAddr)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, c.tlsConfig)
	proxy.SetTCPFastOpen(tlsConn)

	b := buf.NewPacket()
	defer b.Release()

	b.Write(c.passwordHash)
	var paddingLen int
	if pad := padding.DefaultPaddingFactory.Load().GenerateRecordPayloadSizes(0); len(pad) > 0 {
		paddingLen = pad[0]
	}
	binary.BigEndian.PutUint16(b.Extend(2), uint16(paddingLen))
	if paddingLen > 0 {
		b.WriteZeroN(paddingLen)
	}

	if _, err := b.WriteTo(tlsConn); err != nil {
		tlsConn.Close()
		return nil, err
	}

	return tlsConn, nil
}

// NewConnection implements the MistCore TCPConnectionHandler interface for
// use with SOCKS/HTTP inbound handlers.
func (c *Client) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	proxyC, err := c.DialStream(ctx, metadata.Destination)
	if err != nil {
		c.opts.Logger.Errorf("CreateProxy: %v", err)
		return err
	}
	defer proxyC.Close()

	return bufio.CopyConn(ctx, conn, proxyC)
}

// NewPacketConnection implements the MistCore UDPConnectionHandler interface
// for use with SOCKS/HTTP inbound handlers.
func (c *Client) NewPacketConnection(ctx context.Context, conn network.PacketConn, metadata M.Metadata) error {
	proxyC, err := c.DialStream(ctx, uot.RequestDestination(2))
	if err != nil {
		c.opts.Logger.Errorf("CreateProxy: %v", err)
		return err
	}
	defer proxyC.Close()

	request := uot.Request{
		Destination: metadata.Destination,
	}
	uotC := uot.NewLazyConn(proxyC, request)

	return bufio.CopyPacketConn(ctx, conn, uotC)
}

// compile-time check
var _ = util.ProgramVersionName
