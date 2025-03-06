package nobl9repo

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

func newDataCache() *dataCache {
	return &dataCache{
		retention: 5 * time.Minute,
		cache:     make(map[string]cacheEntry),
	}
}

type dataCache struct {
	retention time.Duration
	cache     map[string]cacheEntry
	mu        sync.RWMutex
}

type cacheEntry struct {
	data      any
	expiresAt time.Time
}

func (c *dataCache) Get(ctx context.Context, key string) (data any, hit bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.cache[key]
	if !ok {
		slog.DebugContext(ctx, "cache miss", slog.String("key", key))
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		slog.DebugContext(ctx, "cached entry expired", slog.String("key", key))
		return nil, false
	}
	slog.DebugContext(ctx, "cache hit",
		slog.String("key", key),
		slog.String("expiresAt", entry.expiresAt.String()))
	return entry.data, true
}

func (c *dataCache) Put(key string, data any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.retention),
	}
}
