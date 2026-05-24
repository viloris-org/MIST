package http

import (
	std_bufio "bufio"
	"context"
	"io"
	"net"
	"net/http"
	"strings"

	"MistCore/common"
	"MistCore/common/atomic"
	"MistCore/common/auth"
	"MistCore/common/buf"
	"MistCore/common/bufio"
	E "MistCore/common/exceptions"
	F "MistCore/common/format"
	M "MistCore/common/metadata"
	N "MistCore/common/network"
	"MistCore/common/pipe"
)

type Handler = N.TCPConnectionHandler

func HandleConnection(ctx context.Context, conn net.Conn, reader *std_bufio.Reader, authenticator *auth.Authenticator, handler Handler, metadata M.Metadata) error {
	for {
		request, err := ReadRequest(reader)
		if err != nil {
			return E.Cause(err, "read http request")
		}
		if authenticator != nil {
			var (
				username string
				password string
				authOk   bool
			)
			authorization := request.Header.Get("Proxy-Authorization")
			username, password, authOk = ParseBasicAuth(authorization)
			if authOk {
				authOk = authenticator.Verify(username, password)
				if authOk {
					ctx = auth.ContextWithUser(ctx, username)
				}
			}
			if !authOk {
				err = responseWith(
					request, http.StatusProxyAuthRequired,
					"Proxy-Authenticate", `Basic realm="sing-box" charset="UTF-8"`,
				).Write(conn)
				if err != nil {
					return err
				}
				return E.New("http: authentication failed")
			}
		}

		if sourceAddress := SourceAddress(request); sourceAddress.IsValid() {
			metadata.Source = sourceAddress
		}

		if request.Method == "CONNECT" {
			destination := M.ParseSocksaddrHostPortStr(request.URL.Hostname(), request.URL.Port())
			if destination.Port == 0 {
				switch request.URL.Scheme {
				case "https", "wss":
					destination.Port = 443
				default:
					destination.Port = 80
				}
			}
			_, err = conn.Write([]byte(F.ToString("HTTP/", request.ProtoMajor, ".", request.ProtoMinor, " 200 Connection established\r\n\r\n")))
			if err != nil {
				return E.Cause(err, "write http response")
			}
			metadata.Protocol = "http"
			metadata.Destination = destination

			var requestConn net.Conn
			if reader.Buffered() > 0 {
				buffer := buf.NewSize(reader.Buffered())
				_, err = buffer.ReadFullFrom(reader, reader.Buffered())
				if err != nil {
					return err
				}
				requestConn = bufio.NewCachedConn(conn, buffer)
			} else {
				requestConn = conn
			}
			return handler.NewConnection(ctx, requestConn, metadata)
		} else if strings.ToLower(request.Header.Get("Connection")) == "upgrade" {
			destination := M.ParseSocksaddrHostPortStr(request.URL.Hostname(), request.URL.Port())
			if destination.Port == 0 {
				switch request.URL.Scheme {
				case "https", "wss":
					destination.Port = 443
				default:
					destination.Port = 80
				}
			}
			metadata.Protocol = "http"
			metadata.Destination = destination
			serverConn, clientConn := pipe.Pipe()
			go func() {
				err := handler.NewConnection(ctx, clientConn, metadata)
				if err != nil {
					common.Close(serverConn, clientConn)
				}
			}()
			err = request.Write(serverConn)
			if err != nil {
				return E.Cause(err, "http: write upgrade request")
			}
			if reader.Buffered() > 0 {
				_, err = io.CopyN(serverConn, reader, int64(reader.Buffered()))
				if err != nil {
					return err
				}
			}
			return bufio.CopyConn(ctx, conn, serverConn)
		} else {
			err = handleHTTPConnection(ctx, handler, conn, request, metadata)
			if err != nil {
				return err
			}
		}
	}
}

func handleHTTPConnection(
	ctx context.Context,
	//nolint:staticcheck
	handler N.TCPConnectionHandler,
	conn net.Conn,
	request *http.Request,
	metadata M.Metadata,
) error {
	keepAlive := shouldKeepAlive(request)
	request.RequestURI = ""

	removeHopByHopHeaders(request.Header)
	removeExtraHTTPHostPort(request)

	if hostStr := request.Header.Get("Host"); hostStr != "" {
		if hostStr != request.URL.Host {
			request.Host = hostStr
		}
	}

	if request.URL.Scheme == "" || request.URL.Host == "" {
		return responseWith(request, http.StatusBadRequest).Write(conn)
	}

	var innerErr atomic.TypedValue[error]
	httpClient := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				metadata.Destination = M.ParseSocksaddr(address)
				metadata.Protocol = "http"
				input, output := pipe.Pipe()
				go func() {
					hErr := handler.NewConnection(ctx, output, metadata)
					if hErr != nil {
						innerErr.Store(hErr)
						common.Close(input, output)
					}
				}()
				return input, nil
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	requestCtx, cancel := context.WithCancel(ctx)
	response, err := httpClient.Do(request.WithContext(requestCtx))
	if err != nil {
		cancel()
		return E.Errors(innerErr.Load(), err, responseWith(request, http.StatusBadGateway).Write(conn))
	}

	removeHopByHopHeaders(response.Header)

	if keepAlive {
		response.Header.Set("Proxy-Connection", "keep-alive")
		response.Header.Set("Connection", "keep-alive")
		response.Header.Set("Keep-Alive", "timeout=4")
	}

	response.Close = !keepAlive

	err = response.Write(conn)
	if err != nil {
		cancel()
		return E.Errors(innerErr.Load(), err)
	}

	cancel()
	if !keepAlive {
		return conn.Close()
	}
	return nil
}

func removeHopByHopHeaders(header http.Header) {
	header.Del("Proxy-Connection")
	header.Del("Proxy-Authenticate")
	header.Del("Proxy-Authorization")
	header.Del("TE")
	header.Del("Trailers")
	header.Del("Transfer-Encoding")
	header.Del("Upgrade")

	connections := header.Get("Connection")
	header.Del("Connection")
	if len(connections) == 0 {
		return
	}
	for _, h := range strings.Split(connections, ",") {
		header.Del(strings.TrimSpace(h))
	}
}

func removeExtraHTTPHostPort(req *http.Request) {
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	if pHost, port, err := net.SplitHostPort(host); err == nil && port == "80" {
		if M.ParseAddr(pHost).Is6() {
			pHost = "[" + pHost + "]"
		}
		host = pHost
	}

	req.Host = host
	req.URL.Host = host
}

func shouldKeepAlive(request *http.Request) bool {
	connection := strings.ToLower(strings.TrimSpace(request.Header.Get("Connection")))
	if request.ProtoMajor > 1 || (request.ProtoMajor == 1 && request.ProtoMinor >= 1) {
		return connection != "close"
	}
	proxyConnection := strings.ToLower(strings.TrimSpace(request.Header.Get("Proxy-Connection")))
	return connection == "keep-alive" || proxyConnection == "keep-alive"
}

func responseWith(request *http.Request, statusCode int, headers ...string) *http.Response {
	var header http.Header
	if len(headers) > 0 {
		header = make(http.Header)
		for i := 0; i < len(headers); i += 2 {
			header.Add(headers[i], headers[i+1])
		}
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Proto:      request.Proto,
		ProtoMajor: request.ProtoMajor,
		ProtoMinor: request.ProtoMinor,
		Header:     header,
	}
}
