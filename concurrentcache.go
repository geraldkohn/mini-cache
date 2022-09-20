package cache

import (
	"sync"

	"mini-cache/lru"
)

// 实现并发控制

type cache struct {
	mu            *sync.RWMutex
	lru           *lru.Cache
	cacheMaxBytes int64
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 没有初始化需要初始化
	// 延迟初始化，可以减少内存占用，提高性能。
	if c.lru == nil {
		c.lru = lru.New(c.cacheMaxBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), true
	}

	return
}
