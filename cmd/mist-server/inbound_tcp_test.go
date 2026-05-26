package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	"MistCore/common/buf"
)

func TestTryLegacyAuthRejectsIncompletePadding(t *testing.T) {
	passwordHash := bytes.Repeat([]byte{0x1}, 32)
	b := buf.NewPacket()
	defer b.Release()

	b.Write(passwordHash)
	binary.BigEndian.PutUint16(b.Extend(2), 4)
	b.Write([]byte{0xaa, 0xbb})

	if tryLegacyAuth(b, passwordHash) {
		t.Fatal("expected incomplete padding to fail auth")
	}
}

func TestTryLegacyAuthConsumesCompletePadding(t *testing.T) {
	passwordHash := bytes.Repeat([]byte{0x2}, 32)
	b := buf.NewPacket()
	defer b.Release()

	b.Write(passwordHash)
	binary.BigEndian.PutUint16(b.Extend(2), 2)
	b.Write([]byte{0xaa, 0xbb})

	if !tryLegacyAuth(b, passwordHash) {
		t.Fatal("expected complete legacy auth to succeed")
	}
	if got := len(b.Bytes()); got != 0 {
		t.Fatalf("remaining bytes = %d, want 0", got)
	}
}
