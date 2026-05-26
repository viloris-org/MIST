package padding

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"MistCore/common/atomic"
	"mist/util"
)

const CheckMark = -1

var globalRandomSource = newRandomSource()

type randomSource struct {
	mu    sync.Mutex
	state uint64
}

func newRandomSource() *randomSource {
	var seed [8]byte
	if _, err := rand.Read(seed[:]); err != nil {
		binary.BigEndian.PutUint64(seed[:], uint64(time.Now().UnixNano()))
	}
	return &randomSource{state: binary.BigEndian.Uint64(seed[:])}
}

func (r *randomSource) Uint64() (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.nextLocked(), nil
}

func (r *randomSource) Fill(dst []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for len(dst) >= 8 {
		binary.BigEndian.PutUint64(dst[:8], r.nextLocked())
		dst = dst[8:]
	}
	if len(dst) > 0 {
		var tail [8]byte
		binary.BigEndian.PutUint64(tail[:], r.nextLocked())
		copy(dst, tail[:])
	}
	return nil
}

func (r *randomSource) nextLocked() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

const (
	MaxPaddingSchemeSize = 64 * 1024
	MaxRecordPayloadSize = 16 * 1024
	maxPaddingStop       = 1024
	maxRulesPerRecord    = 64
)

const (
	ProfileRandom = "random"
	ProfileWeb    = "web"
	ProfileAPI    = "api"
	ProfileNone   = "none"
)

var defaultPaddingScheme = func() []byte {
	scheme, err := GenerateProfileScheme(ProfileWeb)
	if err != nil {
		// Fallback to a fixed scheme; should never happen.
		return []byte("stop=8\n0=30-30\n1=100-400\n2=400-500,c,500-1000,c,500-1000,c,500-1000,c,500-1000\n3=9-9,500-1000\n4=500-1000\n5=500-1000\n6=500-1000\n7=500-1000")
	}
	return scheme
}()

type PaddingFactory struct {
	RawScheme          []byte
	Stop               uint32
	Md5                string
	recordPayloadSizes map[uint32][]payloadSizeRule
}

type payloadSizeRule struct {
	min       int
	exclusive int
	checkMark bool
}

var DefaultPaddingFactory atomic.TypedValue[*PaddingFactory]

func init() {
	UpdatePaddingScheme(defaultPaddingScheme)
}

func UpdatePaddingScheme(rawScheme []byte) bool {
	if p := NewPaddingFactory(rawScheme); p != nil {
		DefaultPaddingFactory.Store(p)
		return true
	}
	return false
}

func NewPaddingFactory(rawScheme []byte) *PaddingFactory {
	if len(rawScheme) == 0 || len(rawScheme) > MaxPaddingSchemeSize {
		return nil
	}
	rawScheme = append([]byte(nil), rawScheme...)
	p := &PaddingFactory{
		RawScheme: rawScheme,
		Md5:       fmt.Sprintf("%x", md5.Sum(rawScheme)),
	}
	scheme := util.StringMapFromBytes(rawScheme)
	if len(scheme) == 0 {
		return nil
	}
	if stop, err := strconv.Atoi(strings.TrimSpace(scheme["stop"])); err == nil && stop >= 0 && stop <= maxPaddingStop {
		p.Stop = uint32(stop)
	} else {
		return nil
	}
	recordPayloadSizes, ok := parseRecordPayloadSizes(scheme)
	if !ok {
		return nil
	}
	p.recordPayloadSizes = recordPayloadSizes
	return p
}

func (p *PaddingFactory) GenerateRecordPayloadSizes(pkt uint32) ([]int, error) {
	return p.GenerateRecordPayloadSizesInto(pkt, nil)
}

func (p *PaddingFactory) GenerateRecordPayloadSizesInto(pkt uint32, pktSizes []int) ([]int, error) {
	pktSizes = pktSizes[:0]
	for _, rule := range p.recordPayloadSizes[pkt] {
		if rule.checkMark {
			pktSizes = append(pktSizes, CheckMark)
		} else if rule.min == rule.exclusive {
			pktSizes = append(pktSizes, rule.min)
		} else {
			n, err := randomInt(rule.exclusive - rule.min)
			if err != nil {
				return nil, err
			}
			pktSizes = append(pktSizes, rule.min+n)
		}
	}
	return pktSizes, nil
}

func parseRecordPayloadSizes(scheme util.StringMap) (map[uint32][]payloadSizeRule, bool) {
	records := make(map[uint32][]payloadSizeRule, len(scheme))
	for key, value := range scheme {
		pkt, err := strconv.ParseUint(strings.TrimSpace(key), 10, 32)
		if err != nil {
			continue
		}
		for _, sRange := range strings.Split(value, ",") {
			sRange = strings.TrimSpace(sRange)
			if len(records[uint32(pkt)]) >= maxRulesPerRecord {
				return nil, false
			}
			sRangeMinMax := strings.Split(sRange, "-")
			if len(sRangeMinMax) == 2 {
				_min, err := strconv.ParseInt(strings.TrimSpace(sRangeMinMax[0]), 10, 64)
				if err != nil {
					return nil, false
				}
				_max, err := strconv.ParseInt(strings.TrimSpace(sRangeMinMax[1]), 10, 64)
				if err != nil {
					return nil, false
				}
				_min, _max = min(_min, _max), max(_min, _max)
				if _min <= 0 || _max <= 0 || _max > MaxRecordPayloadSize {
					return nil, false
				}
				records[uint32(pkt)] = append(records[uint32(pkt)], payloadSizeRule{
					min:       int(_min),
					exclusive: int(_max),
				})
			} else if sRange == "c" {
				records[uint32(pkt)] = append(records[uint32(pkt)], payloadSizeRule{checkMark: true})
			} else if sRange != "" {
				return nil, false
			}
		}
	}
	return records, true
}

func randomInt(maxExclusive int) (int, error) {
	span := uint64(maxExclusive)
	limit := ^uint64(0) - (^uint64(0) % span)
	for {
		value, err := globalRandomSource.Uint64()
		if err != nil {
			return 0, err
		}
		if value < limit {
			return int(value % span), nil
		}
	}
}

// RandomInt returns a random integer in [0, maxExclusive). It is safe for
// concurrent use and suitable for infrequent calls outside the hot path.
func RandomInt(maxExclusive int) (int, error) {
	return randomInt(maxExclusive)
}

// FillRandom fills buf with process-seeded pseudo-random bytes. It is safe for
// concurrent use and intended for traffic-shaping padding, not key material.
func FillRandom(buf []byte) error {
	return globalRandomSource.Fill(buf)
}

// GenerateRandomScheme creates a legacy randomized padding scheme.
func GenerateRandomScheme() ([]byte, error) {
	return GenerateProfileScheme(ProfileRandom)
}

// GenerateProfileScheme creates a randomized padding scheme for a traffic
// profile. Packet 0 is kept at 30-30 for preamble compatibility.
func GenerateProfileScheme(profile string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "", ProfileWeb:
		return generateWebScheme()
	case ProfileAPI:
		return generateAPIScheme()
	case ProfileRandom:
		return generateRandomScheme()
	case ProfileNone:
		return generateNoneScheme()
	default:
		return nil, fmt.Errorf("unknown padding profile %q", profile)
	}
}

// ValidateProfile returns an error if profile is not a supported padding
// profile.
func ValidateProfile(profile string) error {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "", ProfileWeb, ProfileAPI, ProfileRandom, ProfileNone:
		return nil
	default:
		return fmt.Errorf("unknown padding profile %q", profile)
	}
}

// generateRandomScheme creates a randomized padding scheme with stop between
// 50 and 200. Remaining packets get randomized rules: some carry checkmarks for
// real data, others are pure waste. Waste sizes vary between 30 and 2000 bytes.
func generateRandomScheme() ([]byte, error) {
	stop, err := randomInt(151) // [0, 150]
	if err != nil {
		return nil, err
	}
	stop += 50 // [50, 200]

	var sb strings.Builder
	fmt.Fprintf(&sb, "stop=%d\n", stop)
	sb.WriteString("0=30-30\n")

	for pkt := 1; pkt < stop; pkt++ {
		fmt.Fprintf(&sb, "%d=", pkt)

		kind, err := randomInt(10)
		if err != nil {
			return nil, err
		}

		switch {
		case kind < 2: // 20%: checkmark only (data passthrough)
			sb.WriteString("c")
		case kind < 5: // 30%: waste + checkmark + waste
			w1, err := randomWasteSize()
			if err != nil {
				return nil, err
			}
			w2, err := randomWasteSize()
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&sb, "%d-%d,c,%d-%d", w1, w1, w2, w2)
		default: // 50%: pure waste, sometimes with a checkmark
			if kind < 6 {
				// single waste chunk
				w, err := randomWasteSize()
				if err != nil {
					return nil, err
				}
				fmt.Fprintf(&sb, "%d-%d", w, w)
			} else if kind < 8 {
				// two waste chunks
				w1, err := randomWasteSize()
				if err != nil {
					return nil, err
				}
				w2, err := randomWasteSize()
				if err != nil {
					return nil, err
				}
				fmt.Fprintf(&sb, "%d-%d,%d-%d", w1, w1, w2, w2)
			} else {
				// waste + occasional checkmark for data passthrough
				w1, err := randomWasteSize()
				if err != nil {
					return nil, err
				}
				w2, err := randomWasteSize()
				if err != nil {
					return nil, err
				}
				fmt.Fprintf(&sb, "%d-%d,c,%d-%d", w1, w1, w2, w2)
			}
		}
		sb.WriteByte('\n')
	}

	return []byte(sb.String()), nil
}

// generateWebScheme biases packet sizes toward a rough HTTPS browsing shape:
// small request/header-like records, medium API/html chunks, and occasional
// larger asset-like chunks. It is still randomized per session.
func generateWebScheme() ([]byte, error) {
	stop, err := randomInt(121) // [0, 120]
	if err != nil {
		return nil, err
	}
	stop += 80 // [80, 200]

	warmupEnd, err := randomInt(7) // [0, 6]
	if err != nil {
		return nil, err
	}
	warmupEnd += 5 // [5, 11]

	burstFracDenom, err := randomInt(3) // [0, 2]
	if err != nil {
		return nil, err
	}
	burstFracDenom += 2 // [2, 4]

	bias, err := randomInt(25) // [0, 24]
	if err != nil {
		return nil, err
	}
	bias -= 12 // [-12, 12]

	var sb strings.Builder
	fmt.Fprintf(&sb, "stop=%d\n", stop)
	sb.WriteString("0=30-30\n")

	for pkt := 1; pkt < stop; pkt++ {
		fmt.Fprintf(&sb, "%d=", pkt)
		phase := webPhase(pkt, stop, warmupEnd, burstFracDenom)
		if err := appendProfileRule(&sb, phase, bias); err != nil {
			return nil, err
		}
		sb.WriteByte('\n')
	}

	return []byte(sb.String()), nil
}

// generateAPIScheme is lower overhead than web while avoiding fixed packet
// sizes. It favours small and medium records with fewer asset-sized chunks.
func generateAPIScheme() ([]byte, error) {
	stop, err := randomInt(81) // [0, 80]
	if err != nil {
		return nil, err
	}
	stop += 40 // [40, 120]

	bias, err := randomInt(25) // [0, 24]
	if err != nil {
		return nil, err
	}
	bias -= 12 // [-12, 12]

	var sb strings.Builder
	fmt.Fprintf(&sb, "stop=%d\n", stop)
	sb.WriteString("0=30-30\n")

	for pkt := 1; pkt < stop; pkt++ {
		fmt.Fprintf(&sb, "%d=", pkt)
		if err := appendProfileRule(&sb, profilePhaseAPI, bias); err != nil {
			return nil, err
		}
		sb.WriteByte('\n')
	}

	return []byte(sb.String()), nil
}

// generateNoneScheme produces a minimal scheme that disables padding after the
// first write. Packet 0 is kept at 30-30 for preamble compatibility; stop=1
// causes sendPadding to become false immediately on the first data write.
func generateNoneScheme() ([]byte, error) {
	return []byte("stop=1\n0=30-30\n"), nil
}

const (
	profilePhaseWarmup = iota
	profilePhaseBurst
	profilePhaseTail
	profilePhaseAPI
)

func webPhase(pkt, stop, warmupEnd, burstFracDenom int) int {
	switch {
	case pkt < warmupEnd:
		return profilePhaseWarmup
	case burstFracDenom > 0 && pkt < stop/burstFracDenom:
		return profilePhaseBurst
	default:
		return profilePhaseTail
	}
}

func appendProfileRule(sb *strings.Builder, phase, bias int) error {
	kind, err := randomInt(100)
	if err != nil {
		return err
	}

	switch phase {
	case profilePhaseWarmup:
		return appendWarmupRule(sb, kind, bias)
	case profilePhaseBurst:
		return appendBurstRule(sb, kind, bias)
	case profilePhaseAPI:
		return appendAPIRule(sb, kind, bias)
	default:
		return appendTailRule(sb, kind, bias)
	}
}

func appendWarmupRule(sb *strings.Builder, kind, bias int) error {
	t1 := clamp(20+bias, 5, 35)
	t2 := clamp(70+bias, 55, 85)
	switch {
	case kind < t1:
		sb.WriteString("c")
	case kind < t2:
		return appendWasteCheckWaste(sb, 40, 360, 40, 900)
	default:
		return appendWasteRange(sb, 300, 1500)
	}
	return nil
}

func appendBurstRule(sb *strings.Builder, kind, bias int) error {
	t1 := clamp(12+bias, 3, 25)
	t2 := clamp(38+bias, 23, 55)
	t3 := clamp(82+bias, 65, 92)
	t4 := clamp(96+bias, 88, 99)
	switch {
	case kind < t1:
		sb.WriteString("c")
	case kind < t2:
		return appendWasteCheckWaste(sb, 60, 700, 300, 1500)
	case kind < t3:
		return appendWasteRange(sb, 900, 4096)
	case kind < t4:
		return appendWasteRange(sb, 4096, MaxRecordPayloadSize)
	default:
		return appendTwoWasteRanges(sb, 300, 1500, 900, 4096)
	}
	return nil
}

func appendTailRule(sb *strings.Builder, kind, bias int) error {
	t1 := clamp(28+bias, 13, 43)
	t2 := clamp(58+bias, 43, 73)
	t3 := clamp(88+bias, 73, 95)
	switch {
	case kind < t1:
		sb.WriteString("c")
	case kind < t2:
		return appendWasteCheckWaste(sb, 40, 220, 40, 700)
	case kind < t3:
		return appendWasteRange(sb, 120, 1200)
	default:
		return appendWasteRange(sb, 1200, 4096)
	}
	return nil
}

func appendAPIRule(sb *strings.Builder, kind, bias int) error {
	t1 := clamp(30+bias, 15, 45)
	t2 := clamp(70+bias, 55, 85)
	t3 := clamp(95+bias, 85, 99)
	switch {
	case kind < t1:
		sb.WriteString("c")
	case kind < t2:
		return appendWasteCheckWaste(sb, 40, 220, 40, 700)
	case kind < t3:
		return appendWasteRange(sb, 300, 1500)
	default:
		return appendWasteRange(sb, 1500, 4096)
	}
	return nil
}

func appendWasteRange(sb *strings.Builder, minSize, maxSize int) error {
	w, err := randomRange(minSize, maxSize)
	if err != nil {
		return err
	}
	fmt.Fprintf(sb, "%d-%d", w, w)
	return nil
}

func appendTwoWasteRanges(sb *strings.Builder, minA, maxA, minB, maxB int) error {
	if err := appendWasteRange(sb, minA, maxA); err != nil {
		return err
	}
	sb.WriteByte(',')
	return appendWasteRange(sb, minB, maxB)
}

func appendWasteCheckWaste(sb *strings.Builder, minA, maxA, minB, maxB int) error {
	if err := appendWasteRange(sb, minA, maxA); err != nil {
		return err
	}
	sb.WriteString(",c,")
	return appendWasteRange(sb, minB, maxB)
}

func randomRange(minSize, maxSize int) (int, error) {
	if minSize < 1 {
		minSize = 1
	}
	if maxSize > MaxRecordPayloadSize {
		maxSize = MaxRecordPayloadSize
	}
	if maxSize < minSize {
		maxSize = minSize
	}
	n, err := randomInt(maxSize - minSize + 1)
	if err != nil {
		return 0, err
	}
	return minSize + n, nil
}

func randomWasteSize() (int, error) {
	n, err := randomInt(1971) // [0, 1970]
	if err != nil {
		return 0, err
	}
	return n + 30, nil // [30, 2000]
}

func clamp(v, lo, hi int) int {
	return max(lo, min(hi, v))
}
