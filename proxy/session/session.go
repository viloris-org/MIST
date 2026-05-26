package session

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"MistCore/common/atomic"
	"MistCore/common/buf"
	"github.com/sirupsen/logrus"
	"mist/proxy/padding"
	"mist/util"
)

const (
	hmacTrailerLen      = 32
	effectiveMaxPayload = maxFramePayloadLen - hmacTrailerLen
	maxCoalescedWrite   = 256 * 1024
)

var clientDebugPaddingScheme = os.Getenv("CLIENT_DEBUG_PADDING_SCHEME") == "1"

var cachedNow atomic.Value

func init() {
	cachedNow.Store(time.Now())
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for range ticker.C {
			cachedNow.Store(time.Now())
		}
	}()
}

func now() time.Time {
	return cachedNow.Load().(time.Time)
}

var errFramePayloadTooLarge = errors.New("frame payload too large")

type Session struct {
	conn    net.Conn
	writeMu sync.Mutex

	streams    map[uint32]*Stream
	streamId   atomic.Uint32
	streamLock sync.RWMutex

	dieOnce sync.Once
	die     chan struct{}
	dieHook func()
	closed  atomic.Bool

	synDone     func()
	synDoneLock sync.Mutex

	// pool
	seq          uint64
	createdAt    time.Time
	idleSince    time.Time
	idleUntil    time.Time
	idleMu       sync.Mutex
	padding      *atomic.TypedValue[*padding.PaddingFactory]
	peerVersion  byte
	frameBufPool sync.Pool

	// client
	isClient           bool
	sendPadding        bool
	buffering          bool
	buffer             []byte
	paddingWriteBuffer []byte
	paddingBuf         []int
	pktCounter         atomic.Uint32

	// server
	onNewStream func(stream *Stream)

	// hardening
	maxStreams        int
	streamBufferSize  int
	readTimeout       time.Duration
	keepaliveInterval time.Duration
	lastRecv          time.Time

	// v3 security
	passwordHash     []byte
	settingsReceived chan struct{}
	authComplete     chan struct{}
	nonce            []byte
	hmacKey          []byte
	hmacPool         sync.Pool
	hmacMode         bool
	authHash         []byte
	pendingPing      bool

	// SYN rate limit
	synCount       int
	synWindowStart time.Time
	synRateLimit   int
}

// SessionInfo exposes read-only session metadata for the dashboard.
type SessionInfo struct {
	Seq         uint64 `json:"seq"`
	StreamCount int    `json:"stream_count"`
	PacketCount uint32 `json:"packet_count"`
	AgeMs       int64  `json:"age_ms"`
	IsIdle      bool   `json:"is_idle"`
	IsClosed    bool   `json:"is_closed"`
}

// Info returns a read-only snapshot of session metadata.
func (s *Session) Info() SessionInfo {
	s.streamLock.RLock()
	streamCount := len(s.streams)
	s.streamLock.RUnlock()

	s.idleMu.Lock()
	isIdle := !s.idleUntil.IsZero() && time.Now().Before(s.idleUntil)
	s.idleMu.Unlock()

	return SessionInfo{
		Seq:         s.seq,
		StreamCount: streamCount,
		PacketCount: s.pktCounter.Load(),
		AgeMs:       time.Since(s.createdAt).Milliseconds(),
		IsIdle:      isIdle,
		IsClosed:    s.IsClosed(),
	}
}

func NewClientSession(conn net.Conn, _padding *atomic.TypedValue[*padding.PaddingFactory], maxStreams int, streamBufferSize int, readTimeout, keepaliveInterval time.Duration, synRateLimit int, passwordHash []byte) *Session {
	s := &Session{
		conn:              conn,
		isClient:          true,
		sendPadding:       true,
		padding:           _padding,
		createdAt:         time.Now(),
		maxStreams:        maxStreams,
		streamBufferSize:  streamBufferSize,
		readTimeout:       readTimeout,
		keepaliveInterval: keepaliveInterval,
		synRateLimit:      synRateLimit,
		passwordHash:      passwordHash,
		settingsReceived:  make(chan struct{}),
	}
	s.frameBufPool = sync.Pool{
		New: func() any {
			buf := make([]byte, headerOverHeadSize+maxFramePayloadLen+hmacTrailerLen)
			return &buf
		},
	}
	s.die = make(chan struct{})
	s.streams = make(map[uint32]*Stream)
	return s
}

func NewServerSession(conn net.Conn, onNewStream func(stream *Stream), _padding *atomic.TypedValue[*padding.PaddingFactory], maxStreams int, streamBufferSize int, readTimeout, keepaliveInterval time.Duration, synRateLimit int, passwordHash []byte) *Session {
	s := &Session{
		conn:              conn,
		onNewStream:       onNewStream,
		padding:           _padding,
		createdAt:         time.Now(),
		maxStreams:        maxStreams,
		streamBufferSize:  streamBufferSize,
		readTimeout:       readTimeout,
		keepaliveInterval: keepaliveInterval,
		synRateLimit:      synRateLimit,
		passwordHash:      passwordHash,
		authComplete:      make(chan struct{}),
	}
	s.frameBufPool = sync.Pool{
		New: func() any {
			buf := make([]byte, headerOverHeadSize+maxFramePayloadLen+hmacTrailerLen)
			return &buf
		},
	}
	s.die = make(chan struct{})
	s.streams = make(map[uint32]*Stream)
	return s
}

func (s *Session) Run() {
	if s.keepaliveInterval > 0 {
		go s.keepaliveLoop()
	}

	if !s.isClient {
		s.recvLoop()
		return
	}

	settings := util.StringMap{
		"v":           "3",
		"client":      util.ProgramVersionName,
		"padding-md5": s.padding.Load().Md5,
	}
	f := newFrame(cmdSettings, 0)
	f.data = settings.ToBytes()
	s.buffering = true
	s.writeControlFrame(f)

	go s.recvLoop()
}

// IsClosed does a safe check to see if we have shutdown
func (s *Session) IsClosed() bool {
	return s.closed.Load()
}

// Close is used to close the session and all streams.
func (s *Session) Close() error {
	var once bool
	s.dieOnce.Do(func() {
		s.closed.Store(true)
		close(s.die)
		once = true
	})
	if once {
		if s.dieHook != nil {
			s.dieHook()
			s.dieHook = nil
		}
		s.streamLock.Lock()
		for _, stream := range s.streams {
			stream.closeLocally()
		}
		s.streams = make(map[uint32]*Stream)
		s.streamLock.Unlock()
		return s.conn.Close()
	} else {
		return io.ErrClosedPipe
	}
}

// OpenStream is used to create a new stream for CLIENT
func (s *Session) OpenStream() (*Stream, error) {
	if s.IsClosed() {
		return nil, io.ErrClosedPipe
	}

	// v3 client: wait for server settings (nonce) before opening streams
	if s.settingsReceived != nil {
		select {
		case <-s.settingsReceived:
		case <-s.die:
			return nil, io.ErrClosedPipe
		}
	}

	sid := s.streamId.Add(1)
	stream := newStream(sid, s)

	//logrus.Debugln("stream open", sid, s.streams)

	if sid >= 2 && s.peerVersion >= 2 {
		s.synDoneLock.Lock()
		if s.synDone != nil {
			s.synDone()
		}
		s.synDone = util.NewDeadlineWatcher(time.Second*3, func() {
			s.Close()
		})
		s.synDoneLock.Unlock()
	}

	if _, err := s.writeControlFrame(newFrame(cmdSYN, sid)); err != nil {
		return nil, err
	}

	s.buffering = false // proxy Write it's SocksAddr to flush the buffer

	s.streamLock.Lock()
	defer s.streamLock.Unlock()
	select {
	case <-s.die:
		return nil, io.ErrClosedPipe
	default:
		s.streams[sid] = stream
		return stream, nil
	}
}

func (s *Session) keepaliveLoop() {
	ticker := time.NewTicker(s.keepaliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if s.pendingPing {
				logrus.Debugln("keepalive timeout, closing session")
				s.Close()
				return
			}
			s.pendingPing = true
			s.writeControlFrame(newFrame(cmdHeartRequest, 0))
		case <-s.die:
			return
		}
	}
}

func (s *Session) recvLoop() error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorln("[BUG]", r, string(debug.Stack()))
		}
	}()
	defer s.Close()

	var receivedSettingsFromClient bool
	var hdr rawHeader

	for {
		if s.IsClosed() {
			return io.ErrClosedPipe
		}
		// read header first
		if s.readTimeout > 0 {
			s.conn.SetReadDeadline(now().Add(s.readTimeout))
		}
		if _, err := io.ReadFull(s.conn, hdr[:]); err == nil {
			s.lastRecv = now()
			sid := hdr.StreamID()
			switch hdr.Cmd() {
			case cmdPSH:
				buffer, err := s.readFramePayload(hdr)
				if err != nil {
					return err
				}
				if len(buffer) > 0 {
					s.streamLock.RLock()
					stream, ok := s.streams[sid]
					s.streamLock.RUnlock()
					if ok {
						stream.deliverData(buffer)
					} else {
						buf.Put(buffer)
					}
				}
			case cmdSYN: // should be server only
				if err := s.drainFramePayload(hdr); err != nil {
					return err
				}
				if !s.isClient && !receivedSettingsFromClient {
					f := newFrame(cmdAlert, 0)
					f.data = []byte("client did not send its settings")
					s.writeControlFrame(f)
					return nil
				}
				if !s.isClient && s.peerVersion >= 3 && s.authComplete != nil {
					select {
					case <-s.authComplete:
					default:
						f := newFrame(cmdAlert, 0)
						f.data = []byte("not authenticated")
						s.writeControlFrame(f)
						return nil
					}
				}
				if s.synRateLimit > 0 {
					n := now()
					if n.Sub(s.synWindowStart) > time.Second {
						s.synCount = 0
						s.synWindowStart = n
					}
					s.synCount++
					if s.synCount > s.synRateLimit {
						f := newFrame(cmdAlert, 0)
						f.data = []byte("SYN rate exceeded")
						s.writeControlFrame(f)
						return nil
					}
				}
				s.streamLock.Lock()
				if s.maxStreams > 0 && len(s.streams) >= s.maxStreams {
					s.streamLock.Unlock()
					f := newFrame(cmdAlert, 0)
					f.data = []byte("max streams exceeded")
					s.writeControlFrame(f)
					return nil
				}
				if _, ok := s.streams[sid]; !ok {
					stream := newStream(sid, s)
					s.streams[sid] = stream
					go func() {
						if s.onNewStream != nil {
							s.onNewStream(stream)
						} else {
							stream.Close()
						}
					}()
				}
				s.streamLock.Unlock()
			case cmdSYNACK: // should be client only
				s.synDoneLock.Lock()
				if s.synDone != nil {
					s.synDone()
					s.synDone = nil
				}
				s.synDoneLock.Unlock()
				buffer, err := s.readFramePayload(hdr)
				if err != nil {
					return err
				}
				if len(buffer) > 0 {
					s.streamLock.RLock()
					stream, ok := s.streams[sid]
					s.streamLock.RUnlock()
					if ok {
						stream.closeWithError(fmt.Errorf("remote: %s", string(buffer)))
					}
					buf.Put(buffer)
				}
			case cmdFIN:
				if err := s.drainFramePayload(hdr); err != nil {
					return err
				}
				s.streamLock.Lock()
				stream, ok := s.streams[sid]
				delete(s.streams, sid)
				s.streamLock.Unlock()
				if ok {
					stream.closeLocally()
				}
			case cmdWaste:
				if hdr.Length() > 0 {
					if _, err := io.CopyN(io.Discard, s.conn, int64(hdr.Length())); err != nil {
						return err
					}
				}
			case cmdSettings:
				buffer, err := s.readFramePayload(hdr)
				if err != nil {
					return err
				}
				if len(buffer) > 0 {
					if !s.isClient {
						receivedSettingsFromClient = true
						m := util.StringMapFromBytes(buffer)
						paddingF := s.padding.Load()
						if m["padding-md5"] != paddingF.Md5 {
							// logrus.Debugln("remote md5 is", m["padding-md5"])
							f := newFrame(cmdUpdatePaddingScheme, 0)
							f.data = paddingF.RawScheme
							_, err = s.writeControlFrame(f)
							if err != nil {
								buf.Put(buffer)
								return err
							}
						}
						// check client's version
						if v, err := strconv.Atoi(m["v"]); err == nil && v >= 2 {
							s.peerVersion = byte(v)
							serverSettings := util.StringMap{
								"v": strconv.Itoa(v),
							}
							if v >= 3 {
								var nonceBytes [32]byte
								if _, err := rand.Read(nonceBytes[:]); err != nil {
									buf.Put(buffer)
									return err
								}
								s.nonce = nonceBytes[:]
								s.setHMACKey(deriveHMACKey(s.passwordHash, s.nonce))
								authMac := hmac.New(sha256.New, s.nonce)
								authMac.Write(s.passwordHash)
								s.authHash = authMac.Sum(nil)
								serverSettings["nonce"] = hex.EncodeToString(nonceBytes[:])
							}
							f := newFrame(cmdServerSettings, 0)
							f.data = serverSettings.ToBytes()
							_, err = s.writeControlFrame(f)
							if err != nil {
								buf.Put(buffer)
								return err
							}
						}
					}
					buf.Put(buffer)
				}
			case cmdAlert:
				buffer, err := s.readFramePayload(hdr)
				if err != nil {
					return err
				}
				if len(buffer) > 0 {
					if s.isClient {
						logrus.Errorln("[Alert from server]", string(buffer))
					}
					buf.Put(buffer)
				}
				return nil
			case cmdUpdatePaddingScheme:
				if hdr.Length() > 0 {
					rawScheme := make([]byte, int(hdr.Length()))
					if _, err := io.ReadFull(s.conn, rawScheme); err != nil {
						return err
					}
					if s.isClient && !clientDebugPaddingScheme {
						if padding.UpdatePaddingScheme(rawScheme) {
							logrus.Infof("[Update padding succeed] %x\n", md5.Sum(rawScheme))
						} else {
							logrus.Warnf("[Update padding failed] %x\n", md5.Sum(rawScheme))
						}
					}
				}
			case cmdHeartRequest:
				if err := s.drainFramePayload(hdr); err != nil {
					return err
				}
				if _, err := s.writeControlFrame(newFrame(cmdHeartResponse, sid)); err != nil {
					return err
				}
			case cmdHeartResponse:
				if err := s.drainFramePayload(hdr); err != nil {
					return err
				}
				s.pendingPing = false
			case cmdServerSettings:
				buffer, err := s.readFramePayload(hdr)
				if err != nil {
					return err
				}
				if len(buffer) > 0 {
					if s.isClient {
						m := util.StringMapFromBytes(buffer)
						if v, err := strconv.Atoi(m["v"]); err == nil {
							s.peerVersion = byte(v)
							if v >= 3 {
								if nonceHex, ok := m["nonce"]; ok {
									if nonceBytes, err := hex.DecodeString(nonceHex); err == nil && len(nonceBytes) == 32 {
										s.nonce = nonceBytes
										s.setHMACKey(deriveHMACKey(s.passwordHash, s.nonce))
										s.hmacMode = true
										authMac := hmac.New(sha256.New, s.nonce)
										authMac.Write(s.passwordHash)
										f := newFrame(cmdAuthProof, 0)
										f.data = authMac.Sum(nil)
										s.writeControlFrame(f)
									}
								}
							}
						}
						if s.settingsReceived != nil {
							close(s.settingsReceived)
						}
					}
					buf.Put(buffer)
				}
			case cmdAuthProof:
				buffer, err := s.readFramePayload(hdr)
				if err != nil {
					return err
				}
				if !s.isClient {
					var valid bool
					if len(buffer) >= 32 && len(s.authHash) == 32 {
						proof := buffer[:32]
						if subtle.ConstantTimeCompare(proof, s.authHash) == 1 {
							valid = true
						}
					}
					if valid {
						s.hmacMode = true
						if s.authComplete != nil {
							close(s.authComplete)
						}
						logrus.Debugln("v3 auth proof validated")
					} else {
						if len(buffer) > 0 {
							buf.Put(buffer)
						}
						f := newFrame(cmdAlert, 0)
						f.data = []byte("authentication failed")
						s.writeControlFrame(f)
						return errors.New("v3 auth failed")
					}
				}
				if len(buffer) > 0 {
					buf.Put(buffer)
				}
			default:
				// I don't know what command it is (can't have data)
			}
		} else {
			return err
		}
	}
}

func (s *Session) streamClosed(sid uint32) error {
	if s.IsClosed() {
		return io.ErrClosedPipe
	}
	_, err := s.writeControlFrame(newFrame(cmdFIN, sid))
	s.streamLock.Lock()
	delete(s.streams, sid)
	s.streamLock.Unlock()
	return err
}

func (s *Session) writeDataFrame(sid uint32, data []byte) (int, error) {
	// HMAC-only path: coalesce frames like the fast path to reduce conn.Write calls.
	if s.hmacMode && !s.sendPadding {
		return s.writeDataFrameHMAC(sid, data)
	}

	// Padding path (with or without HMAC): per-frame writes preserve pktCounter.
	if s.sendPadding {
		return s.writeDataFramePadded(sid, data)
	}

	// Fast path: no padding, no HMAC — coalesce frames into a pooled buffer.
	return s.writeDataFrameFast(sid, data)
}

func (s *Session) writeDataFrameHMAC(sid uint32, data []byte) (int, error) {
	total := 0
	remaining := data
	for len(remaining) > 0 {
		payloadLimit := min(len(remaining), maxCoalescedWrite)

		// Count chunks and compute total batch size.
		chunks := 0
		batchSize := 0
		tmp := remaining[:payloadLimit]
		for len(tmp) > 0 {
			chunkLen := min(len(tmp), effectiveMaxPayload)
			batchSize += chunkLen + headerOverHeadSize + hmacTrailerLen
			tmp = tmp[chunkLen:]
			chunks++
		}

		// Single frame: bypass intermediate copy.
		if chunks == 1 {
			chunkLen := min(payloadLimit, effectiveMaxPayload)
			bufPtr := s.buildFrame(cmdPSH, sid, remaining[:chunkLen])
			s.writeMu.Lock()
			n, err := s.writeConnLocked(*bufPtr)
			s.writeMu.Unlock()
			s.frameBufPool.Put(bufPtr)

			n -= headerOverHeadSize + hmacTrailerLen
			if n < 0 {
				n = 0
			}
			if n > chunkLen {
				n = chunkLen
			}
			total += n
			if err != nil {
				return total, err
			}
			remaining = remaining[chunkLen:]
			continue
		}

		// Multiple frames: build and coalesce into one buffer.
		batchPtr := s.frameBufPool.Get().(*[]byte)
		batchBuf := *batchPtr
		if cap(batchBuf) < batchSize {
			batchBuf = make([]byte, batchSize)
			*batchPtr = batchBuf
		}
		batchBuf = batchBuf[:0]
		batch := remaining[:payloadLimit]
		for len(batch) > 0 {
			chunkLen := min(len(batch), effectiveMaxPayload)
			bufPtr := s.buildFrame(cmdPSH, sid, batch[:chunkLen])
			batchBuf = append(batchBuf, *bufPtr...)
			s.frameBufPool.Put(bufPtr)
			batch = batch[chunkLen:]
		}

		s.writeMu.Lock()
		n, err := s.writeConnLocked(batchBuf)
		s.writeMu.Unlock()
		s.frameBufPool.Put(batchPtr)

		n -= chunks * (headerOverHeadSize + hmacTrailerLen)
		if n < 0 {
			n = 0
		}
		if n > payloadLimit {
			n = payloadLimit
		}
		total += n
		if err != nil {
			return total, err
		}
		remaining = remaining[payloadLimit:]
	}
	return total, nil
}

func (s *Session) writeDataFramePadded(sid uint32, data []byte) (int, error) {
	chunkMax := maxFramePayloadLen
	if s.hmacMode {
		chunkMax = effectiveMaxPayload
	}
	total := 0
	for len(data) > 0 {
		chunkLen := min(len(data), chunkMax)
		if err := s.writePSHFrame(sid, data[:chunkLen]); err != nil {
			return total, err
		}
		total += chunkLen
		data = data[chunkLen:]
	}
	return total, nil
}

func (s *Session) writeDataFrameFast(sid uint32, data []byte) (int, error) {
	total := 0
	remaining := data
	for len(remaining) > 0 {
		payloadLimit := min(len(remaining), maxCoalescedWrite)
		chunks := (payloadLimit + maxFramePayloadLen - 1) / maxFramePayloadLen
		totalFrameLen := payloadLimit + chunks*headerOverHeadSize

		bufPtr := s.frameBufPool.Get().(*[]byte)
		buf := *bufPtr
		if cap(buf) < totalFrameLen {
			buf = make([]byte, totalFrameLen)
			*bufPtr = buf
		}
		frame := buf[:totalFrameLen]
		offset := 0
		batch := remaining[:payloadLimit]
		for len(batch) > 0 {
			chunkLen := min(len(batch), maxFramePayloadLen)
			frame[offset] = cmdPSH
			binary.BigEndian.PutUint32(frame[offset+1:], sid)
			binary.BigEndian.PutUint16(frame[offset+5:], uint16(chunkLen))
			copy(frame[offset+headerOverHeadSize:], batch[:chunkLen])
			offset += headerOverHeadSize + chunkLen
			batch = batch[chunkLen:]
		}

		s.writeMu.Lock()
		n, err := s.writeConnLocked(frame)
		s.writeMu.Unlock()
		s.frameBufPool.Put(bufPtr)

		if n > chunks*headerOverHeadSize {
			n -= chunks * headerOverHeadSize
		} else {
			n = 0
		}
		if n > payloadLimit {
			n = payloadLimit
		}
		total += n
		if err != nil {
			return total, err
		}
		remaining = remaining[payloadLimit:]
	}
	return total, nil
}

func (s *Session) writePSHFrame(sid uint32, data []byte) error {
	_, err := s.writeFrame(cmdPSH, sid, data)
	return err
}

func (s *Session) writeControlFrame(frame frame) (int, error) {
	dataLen := len(frame.data)
	if dataLen > maxFramePayloadLen {
		return 0, errFramePayloadTooLarge
	}

	s.conn.SetWriteDeadline(time.Now().Add(time.Second * 5))

	_, err := s.writeFrame(frame.cmd, frame.sid, frame.data)
	if err != nil {
		s.Close()
		return 0, err
	}

	s.conn.SetWriteDeadline(time.Time{})

	return dataLen, nil
}

// buildFrame builds a single framed packet into a pooled buffer.
// The returned *[]byte must be returned to s.frameBufPool after use.
func (s *Session) buildFrame(cmd byte, sid uint32, data []byte) *[]byte {
	hmacNeeded := s.hmacMode && !isHandshakeFrame(cmd)
	trailerLen := 0
	if hmacNeeded {
		trailerLen = hmacTrailerLen
	}

	frameLen := len(data) + trailerLen + headerOverHeadSize

	bufPtr := s.frameBufPool.Get().(*[]byte)
	buf := *bufPtr
	if cap(buf) < frameLen {
		buf = make([]byte, frameLen)
		*bufPtr = buf
	}
	frame := buf[:frameLen]
	frame[0] = cmd
	binary.BigEndian.PutUint32(frame[1:5], sid)
	binary.BigEndian.PutUint16(frame[5:7], uint16(len(data)+trailerLen))
	copy(frame[headerOverHeadSize:], data)
	if hmacNeeded {
		s.appendFrameHMAC(frame[headerOverHeadSize+len(data):], cmd, sid, data)
	}
	return bufPtr
}

func (s *Session) writeFrame(cmd byte, sid uint32, data []byte) (int, error) {
	hmacNeeded := s.hmacMode && !isHandshakeFrame(cmd)
	trailerLen := 0
	if hmacNeeded {
		trailerLen = hmacTrailerLen
	}

	wirePayloadLen := len(data) + trailerLen
	frameLen := wirePayloadLen + headerOverHeadSize

	bufPtr := s.frameBufPool.Get().(*[]byte)
	buf := *bufPtr
	if cap(buf) < frameLen {
		buf = make([]byte, frameLen)
		*bufPtr = buf
	}
	frame := buf[:frameLen]
	frame[0] = cmd
	binary.BigEndian.PutUint32(frame[1:5], sid)
	binary.BigEndian.PutUint16(frame[5:7], uint16(wirePayloadLen))
	copy(frame[headerOverHeadSize:], data)
	if hmacNeeded {
		s.appendFrameHMAC(frame[headerOverHeadSize+len(data):], cmd, sid, data)
	}

	n, err := s.writeConnLocked(frame)
	s.frameBufPool.Put(bufPtr)

	if n > headerOverHeadSize {
		n -= headerOverHeadSize
	} else {
		n = 0
	}
	if n > len(data) {
		n = len(data)
	}
	return n, err
}

func (s *Session) writeConn(b []byte) (n int, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	return s.writeConnLocked(b)
}

func (s *Session) writeConnLocked(b []byte) (n int, err error) {
	if s.buffering {
		s.buffer = append(s.buffer, b...)
		return len(b), nil
	} else if len(s.buffer) > 0 {
		s.buffer = append(s.buffer, b...)
		b = s.buffer
		s.buffer = nil
	}

	// calulate & send padding
	if s.sendPadding {
		pkt := s.pktCounter.Add(1)
		paddingF := s.padding.Load()
		if pkt < paddingF.Stop {
			pktSizes, genErr := paddingF.GenerateRecordPayloadSizesInto(pkt, s.paddingBuf)
			if genErr != nil {
				s.sendPadding = false
				return s.conn.Write(b)
			}
			s.paddingBuf = pktSizes

			// Compute exact output size to avoid incremental allocations.
			exactCap := 0
			remainingForCalc := len(b)
			for _, l := range pktSizes {
				if l == padding.CheckMark {
					if remainingForCalc == 0 {
						break
					}
					continue
				}
				if remainingForCalc > l {
					exactCap += l
					remainingForCalc -= l
				} else if remainingForCalc > 0 {
					paddingLen := l - remainingForCalc - headerOverHeadSize
					exactCap += remainingForCalc
					if paddingLen > 0 {
						exactCap += headerOverHeadSize + paddingLen
					}
					remainingForCalc = 0
				} else {
					exactCap += headerOverHeadSize + l
				}
			}
			exactCap += remainingForCalc

			if cap(s.paddingWriteBuffer) < exactCap {
				s.paddingWriteBuffer = make([]byte, 0, exactCap)
			}
			out := s.paddingWriteBuffer[:0]
			payload := b
			for _, l := range pktSizes {
				remainPayloadLen := len(payload)
				if l == padding.CheckMark {
					if remainPayloadLen == 0 {
						break
					} else {
						continue
					}
				}
				// logrus.Debugln(pkt, "write", l, "len", remainPayloadLen, "remain", remainPayloadLen-l)
				if remainPayloadLen > l { // this packet is all payload
					out = append(out, payload[:l]...)
					n += l
					payload = payload[l:]
				} else if remainPayloadLen > 0 { // this packet contains padding and the last part of payload
					paddingLen := l - remainPayloadLen - headerOverHeadSize
					out = append(out, payload...)
					if paddingLen > 0 {
						out = appendWasteFrame(out, paddingLen)
					}
					n += remainPayloadLen
					payload = nil
				} else { // this packet is all padding
					out = appendWasteFrame(out, l)
					payload = nil
				}
			}
			// maybe still remain payload to write
			if len(payload) > 0 {
				out = append(out, payload...)
			}
			s.paddingWriteBuffer = out
			_, err = s.conn.Write(out)
			return n, err
		} else {
			s.sendPadding = false
		}
	}

	return s.conn.Write(b)
}

func appendWasteFrame(dst []byte, paddingLen int) []byte {
	frameLen := headerOverHeadSize + paddingLen
	off := len(dst)
	dst = dst[:off+frameLen]
	frame := dst[off:]
	frame[0] = cmdWaste
	binary.BigEndian.PutUint32(frame[1:5], 0)
	binary.BigEndian.PutUint16(frame[5:7], uint16(paddingLen))
	return dst
}

func (s *Session) setHMACKey(key []byte) {
	s.hmacKey = key
	if len(key) == 0 {
		s.hmacPool = sync.Pool{}
		return
	}
	poolKey := append([]byte(nil), key...)
	s.hmacPool = sync.Pool{
		New: func() any {
			return hmac.New(sha256.New, poolKey)
		},
	}
}

func deriveHMACKey(passwordHash, nonce []byte) []byte {
	h := sha256.New()
	h.Write([]byte("mist-frame-mac-v3"))
	h.Write(passwordHash)
	h.Write(nonce)
	return h.Sum(nil)
}

func isHandshakeFrame(cmd byte) bool {
	switch cmd {
	case cmdSettings, cmdServerSettings, cmdUpdatePaddingScheme, cmdAuthProof:
		return true
	}
	return false
}

func (s *Session) appendFrameHMAC(dst []byte, cmd byte, sid uint32, payload []byte) []byte {
	if len(s.hmacKey) == 0 {
		return dst[:0]
	}
	mac := s.hmacPool.Get().(hash.Hash)
	defer s.hmacPool.Put(mac)
	mac.Reset()
	var prefix [7]byte
	prefix[0] = cmd
	binary.BigEndian.PutUint32(prefix[1:5], sid)
	binary.BigEndian.PutUint16(prefix[5:7], uint16(len(payload)))
	mac.Write(prefix[:])
	mac.Write(payload)
	return mac.Sum(dst[:0])
}

func (s *Session) computeFrameHMAC(dst []byte, cmd byte, sid uint32, payload []byte) int {
	return len(s.appendFrameHMAC(dst[:0], cmd, sid, payload))
}

func (s *Session) drainFramePayload(hdr rawHeader) error {
	wireLen := int(hdr.Length())
	if wireLen == 0 {
		return nil
	}
	if s.hmacMode && !isHandshakeFrame(hdr.Cmd()) {
		buffer := buf.Get(wireLen)
		_, err := io.ReadFull(s.conn, buffer)
		buf.Put(buffer)
		if err != nil {
			return err
		}
		payloadLen := wireLen - hmacTrailerLen
		var macBuf [hmacTrailerLen]byte
		expectedMAC := s.appendFrameHMAC(macBuf[:0], hdr.Cmd(), hdr.StreamID(), buffer[:payloadLen])
		if subtle.ConstantTimeCompare(expectedMAC, buffer[payloadLen:]) != 1 {
			return errors.New("HMAC verification failed")
		}
		return nil
	}
	_, err := io.CopyN(io.Discard, s.conn, int64(wireLen))
	return err
}

func (s *Session) readFramePayload(hdr rawHeader) ([]byte, error) {
	wireLen := int(hdr.Length())
	if wireLen == 0 {
		return nil, nil
	}

	cmd := hdr.Cmd()
	if s.hmacMode && !isHandshakeFrame(cmd) {
		if wireLen < hmacTrailerLen {
			return nil, errors.New("frame too short for HMAC")
		}
		buffer := buf.Get(wireLen)
		if _, err := io.ReadFull(s.conn, buffer); err != nil {
			buf.Put(buffer)
			return nil, err
		}
		payloadLen := wireLen - hmacTrailerLen
		payload := buffer[:payloadLen]
		var macBuf [hmacTrailerLen]byte
		expectedMAC := s.appendFrameHMAC(macBuf[:0], cmd, hdr.StreamID(), payload)
		actualMAC := buffer[payloadLen:]
		if subtle.ConstantTimeCompare(expectedMAC, actualMAC) != 1 {
			buf.Put(buffer)
			return nil, errors.New("HMAC verification failed")
		}
		return payload, nil
	}

	buffer := buf.Get(wireLen)
	if _, err := io.ReadFull(s.conn, buffer); err != nil {
		buf.Put(buffer)
		return nil, err
	}
	return buffer, nil
}
