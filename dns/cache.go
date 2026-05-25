package dns

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

type cacheEntry struct {
	msg     *dns.Msg
	expires time.Time
}

// Cache is a TTL-aware LRU DNS cache keyed by question.
type Cache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
	maxSize int

	// accessOrder tracks insertion order for basic LRU eviction.
	accessOrder []string
}

// NewCache creates a new DNS cache with the given maximum number of entries.
func NewCache(maxSize int) *Cache {
	return &Cache{
		entries:     make(map[string]*cacheEntry),
		maxSize:     maxSize,
	}
}

// Get returns a cached DNS message if it exists and hasn't expired.
func (c *Cache) Get(key string) (*dns.Msg, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expires) {
		delete(c.entries, key)
		return nil, false
	}
	return entry.msg.Copy(), true
}

// Put adds a DNS message to the cache with a TTL derived from the answer records.
func (c *Cache) Put(key string, msg *dns.Msg) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entry if at capacity.
	if len(c.entries) >= c.maxSize && len(c.accessOrder) > 0 {
		oldest := c.accessOrder[0]
		c.accessOrder = c.accessOrder[1:]
		delete(c.entries, oldest)
	}

	ttl := minTTL(msg)
	if ttl == 0 {
		ttl = 60 * time.Second
	}
	c.entries[key] = &cacheEntry{
		msg:     msg.Copy(),
		expires: time.Now().Add(ttl),
	}
	c.accessOrder = append(c.accessOrder, key)
}

// Len returns the current number of cached entries.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func minTTL(msg *dns.Msg) time.Duration {
	var min time.Duration
	for _, rr := range msg.Answer {
		ttl := time.Duration(rr.Header().Ttl) * time.Second
		if min == 0 || ttl < min {
			min = ttl
		}
	}
	for _, rr := range msg.Ns {
		ttl := time.Duration(rr.Header().Ttl) * time.Second
		if min == 0 || ttl < min {
			min = ttl
		}
	}
	return min
}
