package cache

import (
	"sync"
	"time"
)

type URL string
type HashValue struct {
	Body       []byte
	StatusCode int
	ExpiresAt  time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[URL]HashValue
	ttl   time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		items: make(map[URL]HashValue),
		ttl:   ttl,
	}
}

func (c *Cache) Set(url URL, body []byte, statusCode int) {
	c.mu.Lock()
	cpBody := make([]byte, len(body))
	copy(cpBody, body)
	c.items[url] = HashValue{
		Body:       cpBody,
		StatusCode: statusCode,
		ExpiresAt:  time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *Cache) Get(url URL) ([]byte, int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.items[url]
	if !ok {
		return nil, 0, false
	}
	if time.Now().After(v.ExpiresAt) {
		return nil, 0, false
	}

	cpBody := make([]byte, len(v.Body))
	copy(cpBody, v.Body)
	return cpBody, v.StatusCode, true
}
