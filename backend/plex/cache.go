package plex

import (
	"sync"
	"time"
)

type discoveryCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]ServerConn
}

func newDiscoveryCache(ttl time.Duration) *discoveryCache {
	return &discoveryCache{
		ttl:     ttl,
		entries: make(map[string]ServerConn),
	}
}

func (c *discoveryCache) get(userID string) (ServerConn, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, ok := c.entries[userID]
	if !ok {
		return ServerConn{}, false
	}
	if time.Since(conn.ResolvedAt) > c.ttl {
		delete(c.entries, userID)
		return ServerConn{}, false
	}
	return conn, true
}

func (c *discoveryCache) set(userID string, conn ServerConn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn.ResolvedAt.IsZero() {
		conn.ResolvedAt = time.Now()
	}
	c.entries[userID] = conn
}

func (c *discoveryCache) invalidate(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
}
