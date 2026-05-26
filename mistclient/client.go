package mistclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"mist/proxy"
	"mist/proxy/padding"
	"mist/proxy/session"
	"mist/proxy/wsconn"
	"mist/util"

	"MistCore/common/atomic"
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
	padding       *padding.PaddingFactory
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
	rawPadding, err := padding.GenerateProfileScheme(opts.TrafficProfile)
	if err != nil {
		cancel()
		return nil, err
	}
	paddingFactory := padding.NewPaddingFactory(rawPadding)
	if paddingFactory == nil {
		cancel()
		return nil, fmt.Errorf("create padding profile %q", opts.TrafficProfile)
	}
	c := &Client{
		opts:         opts,
		passwordHash: opts.PasswordHash(),
		tlsConfig:    tlsConfig,
		padding:      paddingFactory,
		die:          ctx,
		dieCancel:    cancel,
	}

	var sessionPadding atomic.TypedValue[*padding.PaddingFactory]
	sessionPadding.Store(paddingFactory)

	sessionTimeout := time.Second * 30
	c.sessionClient = session.NewClient(
		ctx,
		c.dialSession,
		&sessionPadding,
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

	if !strings.EqualFold(c.opts.Transport, "wss") {
		if err := c.writeLegacyAuth(tlsConn); err != nil {
			tlsConn.Close()
			return nil, err
		}
		return tlsConn, nil
	}

	b := buf.NewPacket()
	defer b.Release()
	wsKey, err := newWebSocketKey()
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	// Use a real WebSocket upgrade envelope so the long-lived binary stream
	// has a common HTTPS shape instead of a one-off HTTP-looking preamble.
	b.WriteString("GET / HTTP/1.1\r\n")
	fmt.Fprintf(b, "Host: %s\r\n", c.httpHost())
	b.WriteString("Upgrade: websocket\r\n")
	b.WriteString("Connection: Upgrade\r\n")
	fmt.Fprintf(b, "Sec-WebSocket-Key: %s\r\n", wsKey)
	b.WriteString("Sec-WebSocket-Version: 13\r\n")
	fmt.Fprintf(b, "Authorization: Bearer %s\r\n", base64.RawURLEncoding.EncodeToString(c.passwordHash))
	b.WriteString("User-Agent: Mozilla/5.0\r\n")
	b.WriteString("Accept: */*\r\n")
	b.WriteString("\r\n")

	if _, err := b.WriteTo(tlsConn); err != nil {
		tlsConn.Close()
		return nil, err
	}
	if err := readWebSocketUpgrade(tlsConn, wsKey); err != nil {
		tlsConn.Close()
		return nil, err
	}

	return wsconn.NewClient(tlsConn), nil
}

func (c *Client) writeLegacyAuth(tlsConn net.Conn) error {
	b := buf.NewPacket()
	defer b.Release()

	b.Write(c.passwordHash)
	var paddingLen int
	if pad, err := c.padding.GenerateRecordPayloadSizes(0); err == nil && len(pad) > 0 {
		paddingLen = pad[0]
	}
	binary.BigEndian.PutUint16(b.Extend(2), uint16(paddingLen))
	if paddingLen > 0 {
		b.WriteZeroN(paddingLen)
	}
	_, err := b.WriteTo(tlsConn)
	return err
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

func newWebSocketKey() (string, error) {
	var key [16]byte
	if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

func readWebSocketUpgrade(conn net.Conn, wsKey string) error {
	header := make([]byte, 0, 256)
	var one [1]byte
	for !bytes.Contains(header, []byte("\r\n\r\n")) {
		if len(header) > 4096 {
			return fmt.Errorf("websocket upgrade response too large")
		}
		if _, err := io.ReadFull(conn, one[:]); err != nil {
			return err
		}
		header = append(header, one[0])
	}
	if !bytes.HasPrefix(header, []byte("HTTP/1.1 101 ")) && !bytes.HasPrefix(header, []byte("HTTP/1.0 101 ")) {
		lineEnd := bytes.Index(header, []byte("\r\n"))
		if lineEnd < 0 {
			lineEnd = len(header)
		}
		return fmt.Errorf("websocket upgrade failed: %s", string(header[:lineEnd]))
	}
	accept := headerValue(header, "sec-websocket-accept")
	if accept == "" {
		return fmt.Errorf("websocket upgrade missing accept header")
	}
	if accept != webSocketAccept(wsKey) {
		return fmt.Errorf("websocket upgrade accept mismatch")
	}
	return nil
}

func headerValue(header []byte, name string) string {
	for _, line := range bytes.Split(header, []byte("\r\n")) {
		colon := bytes.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(string(line[:colon])), name) {
			return strings.TrimSpace(string(line[colon+1:]))
		}
	}
	return ""
}

func webSocketAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(strings.TrimSpace(key)))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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
