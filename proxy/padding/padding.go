package padding

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"MistCore/common/atomic"
	"mist/util"
)

const CheckMark = -1

const randomBufferSize = 4096

var globalRandomSource = &randomSource{}

type randomSource struct {
	mu  sync.Mutex
	buf [randomBufferSize]byte
	off int
}

func (r *randomSource) Uint64() (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.off+8 > len(r.buf) {
		if _, err := io.ReadFull(rand.Reader, r.buf[:]); err != nil {
			return 0, err
		}
		r.off = 0
	}
	v := binary.BigEndian.Uint64(r.buf[r.off : r.off+8])
	r.off += 8
	return v, nil
}

const (
	MaxPaddingSchemeSize = 64 * 1024
	MaxRecordPayloadSize = 16 * 1024
	maxPaddingStop       = 1024
	maxRulesPerRecord    = 64
)

var defaultPaddingScheme = []byte(`stop=8
0=30-30
1=100-400
2=400-500,c,500-1000,c,500-1000,c,500-1000,c,500-1000
3=9-9,500-1000
4=500-1000
5=500-1000
6=500-1000
7=500-1000`)

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
