package http

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"net"
	stdhttp "net/http"
	"testing"

	"MistCore/common/auth"
	M "MistCore/common/metadata"
)

type testHandler struct {
	ctxCh chan context.Context
}

func (h *testHandler) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	h.ctxCh <- ctx
	_, _ = io.Copy(io.Discard, conn)
	return nil
}

func TestHandleConnectionParsesStandardBasicAuth(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handler := &testHandler{ctxCh: make(chan context.Context, 1)}
	authenticator := auth.NewAuthenticator([]auth.User{{Username: "user", Password: "p+/="}})
	errCh := make(chan error, 1)

	go func() {
		errCh <- HandleConnection(context.Background(), serverConn, bufio.NewReader(serverConn), authenticator, handler, M.Metadata{})
	}()

	request, err := stdhttp.NewRequest(stdhttp.MethodConnect, "http://example.com:443", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:p+/=")))
	if err := request.Write(clientConn); err != nil {
		t.Fatal(err)
	}

	response, err := stdhttp.ReadResponse(bufio.NewReader(clientConn), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}

	ctx := <-handler.ctxCh
	user, ok := auth.UserFromContext[string](ctx)
	if !ok || user != "user" {
		t.Fatalf("unexpected auth context: ok=%v user=%q", ok, user)
	}

	_ = clientConn.Close()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestShouldKeepAlive(t *testing.T) {
	tests := []struct {
		name       string
		protoMajor int
		protoMinor int
		headers    map[string]string
		want       bool
	}{
		{
			name:       "http11 defaults to keep alive",
			protoMajor: 1,
			protoMinor: 1,
			want:       true,
		},
		{
			name:       "http11 close disables keep alive",
			protoMajor: 1,
			protoMinor: 1,
			headers:    map[string]string{"Connection": "close"},
			want:       false,
		},
		{
			name:       "http10 connection keep alive",
			protoMajor: 1,
			protoMinor: 0,
			headers:    map[string]string{"Connection": "keep-alive"},
			want:       true,
		},
		{
			name:       "http10 proxy connection keep alive",
			protoMajor: 1,
			protoMinor: 0,
			headers:    map[string]string{"Proxy-Connection": "keep-alive"},
			want:       true,
		},
		{
			name:       "http10 defaults to close",
			protoMajor: 1,
			protoMinor: 0,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &stdhttp.Request{
				ProtoMajor: tt.protoMajor,
				ProtoMinor: tt.protoMinor,
				Header:     make(stdhttp.Header),
			}
			for key, value := range tt.headers {
				request.Header.Set(key, value)
			}
			if got := shouldKeepAlive(request); got != tt.want {
				t.Fatalf("shouldKeepAlive() = %v, want %v", got, tt.want)
			}
		})
	}
}
