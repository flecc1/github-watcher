package cache

import (
	"sync"
	"time"
)

type URL string
type HashValue struct {
	Body       []byte
	StatusCode int
	TimeOut    time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[URL]HashValue
	ttl   time.Duration
}
