package mistclient

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"mist/proxy"
	"mist/proxy/padding"
	"mist/proxy/session"
	"mist/util"

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

type Stats struct {
	SessionPool session.ClientStats   `json:"session_pool"`
	Sessions    []session.SessionInfo `json:"sessions"`
}

// NewClient creates a new MIST client. It validates the options, fills
// defaults, builds a TLS config, and starts the session pool.
func NewClient(opts Options) (*Client, error) {
	opts.SetDefaults()
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	tlsConfig, err := opts.TLSConfig()
	if err != nil {
		cancel()
		return nil, err
	}
	c := &Client{
		opts:         opts,
		passwordHash: opts.PasswordHash(),
		tlsConfig:    tlsConfig,
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
		opts.StreamBufferSize,
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

func (c *Client) Stats() Stats {
	return Stats{
		SessionPool: c.sessionClient.Stats(),
		Sessions:    c.sessionClient.Sessions(),
	}
}

func (c *Client) dialSession(ctx context.Context) (net.Conn, error) {
	conn, err := proxy.SystemDialer.DialContext(ctx, "tcp", c.opts.ServerAddr)
	if err != nil {
		return nil, err
	}
	proxy.SetTCPFastOpen(conn)

	tlsConn := tls.Client(conn, c.tlsConfig)

	b := buf.NewPacket()
	defer b.Release()

	// Embed the auth hash in a fake HTTP request so the handshake looks
	// like normal web traffic to DPI.
	b.WriteString("GET / HTTP/1.1\r\n")
	fmt.Fprintf(b, "Host: %s\r\n", c.httpHost())
	fmt.Fprintf(b, "Authorization: Bearer %s\r\n", base64.RawURLEncoding.EncodeToString(c.passwordHash))
	b.WriteString("User-Agent: Mozilla/5.0\r\n")
	b.WriteString("Accept: */*\r\n")
	b.WriteString("\r\n")

	// Random HTTP body as preamble padding.
	bodyLen, _ := padding.RandomInt(101) // [0, 100]
	b.WriteRandom(bodyLen + 30)          // [30, 130]

	if _, err := b.WriteTo(tlsConn); err != nil {
		tlsConn.Close()
		return nil, err
	}

	return tlsConn, nil
}

func (c *Client) httpHost() string {
	if host := strings.TrimSpace(c.tlsConfig.ServerName); host != "" {
		return host
	}
	host, _, err := net.SplitHostPort(c.opts.ServerAddr)
	if err == nil && host != "" {
		return host
	}
	return c.opts.ServerAddr
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
