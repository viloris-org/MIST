package session

import (
	"mist/proxy/padding"
	"mist/util"
	"cmp"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"MistCore/common/atomic"
	"github.com/sirupsen/logrus"
)

var clientDebugSessionPool = os.Getenv("CLIENT_DEBUG_SESSION_POOL") == "1"
var clientStreamCounter atomic.Uint64

const sessionPoolJitterPercent = 10

type Client struct {
	die       context.Context
	dieCancel context.CancelFunc

	dialOut util.DialOutFunc

	sessionCounter atomic.Uint64

	idleSession     *idleSessionPool
	idleSessionLock sync.Mutex

	sessions     map[uint64]*Session
	sessionsLock sync.Mutex

	padding *atomic.TypedValue[*padding.PaddingFactory]

	idleSessionTimeout time.Duration
	minIdleSession     int

	maxStreams        int
	readTimeout       time.Duration
	keepaliveInterval time.Duration
	synRateLimit      int
	passwordHash      []byte
}

func NewClient(ctx context.Context, dialOut util.DialOutFunc,
	_padding *atomic.TypedValue[*padding.PaddingFactory], idleSessionCheckInterval, idleSessionTimeout time.Duration, minIdleSession int,
	maxStreams int, readTimeout, keepaliveInterval time.Duration, synRateLimit int, passwordHash []byte,
) *Client {
	c := &Client{
		sessions:           make(map[uint64]*Session),
		dialOut:            dialOut,
		padding:            _padding,
		idleSessionTimeout: idleSessionTimeout,
		minIdleSession:     minIdleSession,
		maxStreams:         maxStreams,
		readTimeout:        readTimeout,
		keepaliveInterval:  keepaliveInterval,
		synRateLimit:       synRateLimit,
		passwordHash:       passwordHash,
	}
	if idleSessionCheckInterval <= time.Second*5 {
		idleSessionCheckInterval = time.Second * 30
	}
	if c.idleSessionTimeout <= time.Second*5 {
		c.idleSessionTimeout = time.Second * 30
	}
	c.die, c.dieCancel = context.WithCancel(ctx)
	c.idleSession = newIdleSessionPool()
	c.startIdleCleanup(idleSessionCheckInterval)
	return c
}

func (c *Client) CreateStream(ctx context.Context) (net.Conn, error) {
	select {
	case <-c.die.Done():
		return nil, io.ErrClosedPipe
	default:
	}

	var session *Session
	var stream *Stream
	var err error

	session = c.getIdleSession()
	if session == nil {
		session, err = c.createSession(ctx)
		if session != nil && clientDebugSessionPool {
			logrus.Infoln("create session:", session.seq)
		}
	} else {
		if clientDebugSessionPool {
			logrus.Infoln("get session:", session.seq)
		}
	}
	if session == nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	stream, err = session.OpenStream()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	if clientDebugSessionPool {
		cn := clientStreamCounter.Add(1)
		s := c.sessionCounter.Load()
		logrus.Infoln("cumulative session:", s, "cumulative stream:", cn, "avg:", float64(cn)/float64(s))
	}

	stream.dieHook = func() {
		// If Session is not closed, put this Stream to pool
		if !session.IsClosed() {
			if clientDebugSessionPool {
				logrus.Infoln("put session:", session.seq, stream.id)
			}
			select {
			case <-c.die.Done():
				// Now client has been closed
				go session.Close()
			default:
				c.idleSessionLock.Lock()
				session.idleSince = time.Now()
				session.idleUntil = session.idleSince.Add(jitterDuration(c.idleSessionTimeout, sessionPoolJitterPercent))
				c.idleSession.Insert(math.MaxUint64-session.seq, session)
				c.idleSessionLock.Unlock()
			}
		} else {
			if clientDebugSessionPool {
				logrus.Infoln("discard session stream:", session.seq, stream.id)
			}
		}
	}

	return stream, nil
}

func (c *Client) getIdleSession() (idle *Session) {
	c.idleSessionLock.Lock()
	idle = c.idleSession.PopFirst()
	c.idleSessionLock.Unlock()
	return
}

func (c *Client) createSession(ctx context.Context) (*Session, error) {
	underlying, err := c.dialOut(ctx)
	if err != nil {
		return nil, err
	}

	session := NewClientSession(underlying, &padding.DefaultPaddingFactory, c.maxStreams, c.readTimeout, c.keepaliveInterval, c.synRateLimit, c.passwordHash)
	session.seq = c.sessionCounter.Add(1)
	session.dieHook = func() {
		if clientDebugSessionPool {
			logrus.Infoln("session died:", session.seq, session.streamId.Load(), session.pktCounter.Load())
		}

		c.idleSessionLock.Lock()
		c.idleSession.Remove(math.MaxUint64 - session.seq)
		c.idleSessionLock.Unlock()

		c.sessionsLock.Lock()
		delete(c.sessions, session.seq)
		c.sessionsLock.Unlock()
	}

	c.sessionsLock.Lock()
	c.sessions[session.seq] = session
	c.sessionsLock.Unlock()

	session.Run()
	return session, nil
}

func (c *Client) Close() error {
	c.dieCancel()

	c.sessionsLock.Lock()
	sessionToClose := make([]*Session, 0, len(c.sessions))
	for _, session := range c.sessions {
		sessionToClose = append(sessionToClose, session)
	}
	c.sessions = make(map[uint64]*Session)
	c.sessionsLock.Unlock()

	for _, session := range sessionToClose {
		session.Close()
	}

	return nil
}

func (c *Client) startIdleCleanup(interval time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorln("[BUG]", r, string(debug.Stack()))
			}
		}()
		for {
			timer := time.NewTimer(jitterDuration(interval, sessionPoolJitterPercent))
			select {
			case <-timer.C:
				c.idleCleanup(time.Now())
			case <-c.die.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			}
		}
	}()
}

func (c *Client) idleCleanup(now time.Time) {
	c.idleCleanupExpTime(now)
}

func (c *Client) idleCleanupExpTime(expTime time.Time) {
	activeCount := 0
	var sessionToClose []*Session

	c.idleSessionLock.Lock()
	var activeItems []idleSessionItem
	for _, item := range c.idleSession.items {
		session := item.session
		if clientDebugSessionPool {
			logrus.Debugln("check session:", session.seq, expTime, session.idleSince, session.idleUntil)
		}

		if session.idleUntil.IsZero() {
			session.idleUntil = session.idleSince.Add(jitterDuration(c.idleSessionTimeout, sessionPoolJitterPercent))
		}
		if session.idleUntil.After(expTime) {
			activeCount++
			activeItems = append(activeItems, item)
			continue
		}

		if activeCount < c.minIdleSession {
			session.idleSince = time.Now()
			session.idleUntil = session.idleSince.Add(jitterDuration(c.idleSessionTimeout, sessionPoolJitterPercent))
			activeCount++
			activeItems = append(activeItems, item)
			continue
		}

		sessionToClose = append(sessionToClose, session)
	}
	c.idleSession.items = activeItems
	c.idleSessionLock.Unlock()

	for _, session := range sessionToClose {
		if clientDebugSessionPool {
			logrus.Infoln("local cleanup session:", session.seq)
		}
		session.Close()
	}
}

func jitterDuration(base time.Duration, percent int) time.Duration {
	if base <= 0 || percent <= 0 {
		return base
	}
	window := int64(base) * int64(percent) / 100
	if window <= 0 {
		return base
	}
	offset := randomInt64(window*2+1) - window
	return base + time.Duration(offset)
}

func randomInt64(maxExclusive int64) int64 {
	span := uint64(maxExclusive)
	limit := ^uint64(0) - (^uint64(0) % span)
	for {
		var randomBytes [8]byte
		if _, err := rand.Read(randomBytes[:]); err != nil {
			return 0
		}
		value := binary.BigEndian.Uint64(randomBytes[:])
		if value < limit {
			return int64(value % span)
		}
	}
}

type idleSessionItem struct {
	key     uint64
	session *Session
}

type idleSessionPool struct {
	items []idleSessionItem
}

func newIdleSessionPool() *idleSessionPool {
	return &idleSessionPool{}
}

func (p *idleSessionPool) Insert(key uint64, session *Session) {
	idx, found := slices.BinarySearchFunc(p.items, key, func(item idleSessionItem, k uint64) int {
		return cmp.Compare(item.key, k)
	})
	if found {
		p.items[idx].session = session
	} else {
		p.items = slices.Insert(p.items, idx, idleSessionItem{key: key, session: session})
	}
}

func (p *idleSessionPool) Remove(key uint64) {
	idx, found := slices.BinarySearchFunc(p.items, key, func(item idleSessionItem, k uint64) int {
		return cmp.Compare(item.key, k)
	})
	if found {
		p.items = slices.Delete(p.items, idx, idx+1)
	}
}

func (p *idleSessionPool) IsEmpty() bool {
	return len(p.items) == 0
}

func (p *idleSessionPool) PopFirst() *Session {
	if len(p.items) == 0 {
		return nil
	}
	s := p.items[0].session
	p.items[0] = idleSessionItem{} // allow GC of popped item
	p.items = p.items[1:]
	return s
}
