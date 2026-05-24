package session

import (
	"math"
	"testing"
	"time"
)

func TestJitterDurationStaysWithinBounds(t *testing.T) {
	base := 30 * time.Second
	minDuration := 27 * time.Second
	maxDuration := 33 * time.Second

	for i := 0; i < 100; i++ {
		got := jitterDuration(base, sessionPoolJitterPercent)
		if got < minDuration || got > maxDuration {
			t.Fatalf("jitterDuration() = %s, want between %s and %s", got, minDuration, maxDuration)
		}
	}
}

func TestIdleCleanupKeepsSessionBeforeIdleUntil(t *testing.T) {
	now := time.Now()
	c := &Client{
		idleSession:        newIdleSessionPool(),
		idleSessionTimeout: 30 * time.Second,
	}
	s := &Session{
		conn:              &recordingConn{},
		die:               make(chan struct{}),
		streams:           make(map[uint32]*Stream),
		seq:               1,
		idleSince:         now.Add(-time.Minute),
		idleUntil:         now.Add(time.Second),
		settingsReceived:  make(chan struct{}),
	}
	c.idleSession.Insert(math.MaxUint64-s.seq, s)

	c.idleCleanupExpTime(now)

	if c.idleSession.IsEmpty() {
		t.Fatal("session was cleaned before idleUntil")
	}
	if s.IsClosed() {
		t.Fatal("session was closed before idleUntil")
	}
}
