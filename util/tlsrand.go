package util

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"io"
)

// tlsRand returns a random uint64 using crypto/rand.
func tlsRandUint64() uint64 {
	var buf [8]byte
	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		// Fallback on catastrophic RNG failure; should never happen.
		return 0
	}
	return binary.BigEndian.Uint64(buf[:])
}

// shuffleCurves shuffles a slice of tls.CurveID deterministically from seed.
func shuffleCurves(curves []tls.CurveID) {
	for i := len(curves) - 1; i > 0; i-- {
		j := int(tlsRandUint64() % uint64(i+1))
		curves[i], curves[j] = curves[j], curves[i]
	}
}

// RandomizedCurvePreferences returns a shuffled list of modern ECDHE curves
// suitable for use as tls.Config.CurvePreferences. P-256 and P-384 are mixed
// with X25519 to avoid a distinctive single-curve fingerprint.
func RandomizedCurvePreferences() []tls.CurveID {
	curves := []tls.CurveID{
		tls.X25519,
		tls.CurveP256,
		tls.CurveP384,
	}
	shuffleCurves(curves)
	return curves
}

// shuffleStrings shuffles a slice of strings deterministically from seed.
func shuffleStrings(s []string) {
	for i := len(s) - 1; i > 0; i-- {
		j := int(tlsRandUint64() % uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}

// RandomizedALPN returns a randomized set of ALPN protocol identifiers.
// Includes common values (h2, http/1.1) to blend with normal TLS traffic.
// The MIST server ignores ALPN, so these are never actually negotiated.
func RandomizedALPN() []string {
	protos := []string{"h2", "http/1.1"}
	shuffleStrings(protos)

	// Occasionally drop one to vary length.
	if tlsRandUint64()%3 == 0 {
		protos = protos[:1]
	}

	return protos
}
