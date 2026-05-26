package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// deriveObfsKey derives a per-session header obfuscation key from the HMAC key.
// Uses a distinct domain-separation prefix to avoid key reuse.
func deriveObfsKey(hmacKey []byte) []byte {
	h := sha256.New()
	h.Write([]byte("mist-header-obfs-v1"))
	h.Write(hmacKey)
	return h.Sum(nil)
}

// xorHeader XORs the 7-byte frame header in-place with a keystream derived from
// obfsKey and counter. The counter must be a frame sequence number that both
// sides derive identically (send counter on write, recv counter on read).
func xorHeader(header []byte, obfsKey []byte, counter uint64) {
	mac := hmac.New(sha256.New, obfsKey)
	var ctr [8]byte
	binary.BigEndian.PutUint64(ctr[:], counter)
	mac.Write(ctr[:])
	keystream := mac.Sum(nil)
	for i := range len(header) {
		header[i] ^= keystream[i]
	}
}
