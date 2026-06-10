package fsub

import (
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	isMember bool
	expiry   time.Time
}

type FsubCache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
}

func NewFsubCache() *FsubCache {
	return &FsubCache{
		items: make(map[string]cacheEntry),
	}
}

func (c *FsubCache) Set(userId int64, channelId int64, isMember bool, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%d:%d", userId, channelId)
	c.items[key] = cacheEntry{
		isMember: isMember,
		expiry:   time.Now().Add(ttl),
	}
}

func (c *FsubCache) Get(userId int64, channelId int64) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := fmt.Sprintf("%d:%d", userId, channelId)
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expiry) {
		return false, false
	}
	return entry.isMember, true
}

// Clear flushes all cached membership entries.
// Must be called whenever Fsub channels are added/removed/changed.
func (c *FsubCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheEntry)
}

// InvalidateChannel removes all cached entries for a specific channel.
func (c *FsubCache) InvalidateChannel(channelId int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	suffix := fmt.Sprintf(":%d", channelId)
	for key := range c.items {
		if len(key) > len(suffix) && key[len(key)-len(suffix):] == suffix {
			delete(c.items, key)
		}
	}
}

// AntiSpamCache tracks users warned in group chats
type AntiSpamCache struct {
	mu    sync.RWMutex
	items map[int64]time.Time
}

func NewAntiSpamCache() *AntiSpamCache {
	return &AntiSpamCache{
		items: make(map[int64]time.Time),
	}
}

func (c *AntiSpamCache) ShouldWarn(userId int64, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	expiry, ok := c.items[userId]
	if !ok || time.Now().After(expiry) {
		c.items[userId] = time.Now().Add(ttl)
		return true
	}
	return false
}
