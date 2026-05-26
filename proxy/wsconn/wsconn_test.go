package wsconn

import (
	"bytes"
	"io"
	"net"
	"testing"
)

func TestConnRoundTrip(t *testing.T) {
	clientRaw, serverRaw := net.Pipe()
	defer clientRaw.Close()
	defer serverRaw.Close()

	client := NewClient(clientRaw)
	server := NewServer(serverRaw)
	payload := bytes.Repeat([]byte("abc123"), 200)

	errCh := make(chan error, 1)
	go func() {
		_, err := client.Write(payload)
		errCh <- err
	}()

	got := make([]byte, len(payload))
	if _, err := io.ReadFull(server, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("server read different payload")
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestConnServerToClientRoundTrip(t *testing.T) {
	clientRaw, serverRaw := net.Pipe()
	defer clientRaw.Close()
	defer serverRaw.Close()

	client := NewClient(clientRaw)
	server := NewServer(serverRaw)
	payload := bytes.Repeat([]byte("xyz"), 100)

	errCh := make(chan error, 1)
	go func() {
		_, err := server.Write(payload)
		errCh <- err
	}()

	got := make([]byte, len(payload))
	if _, err := io.ReadFull(client, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("client read different payload")
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}
