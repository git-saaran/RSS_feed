package cache

import (
	"sync"
	"time"
)

type CacheItem struct {
	Value      interface{}
	Expiration int64
}

type Cache struct {
	items    map[string]CacheItem
	mu       sync.RWMutex
	duration time.Duration
}

func NewCache(duration time.Duration) *Cache {
	c := &Cache{
		items:    make(map[string]CacheItem),
		duration: duration,
	}

	// Start background cleanup
	go c.cleanup()

	return c
}

func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: time.Now().Add(c.duration).UnixNano(),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return nil, false
	}

	if time.Now().UnixNano() > item.Expiration {
		c.Delete(key)
		return nil, false
	}

	return item.Value, true
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UnixNano()
		c.mu.Lock()
		for key, item := range c.items {
			if now > item.Expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
