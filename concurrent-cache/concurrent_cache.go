package concurrentcache

import "mini-cache/view"

type ConcurrentCache struct {
	cacheMaxBytes uint64
	cl            *concurrentList
	cm            concurrentMap
}

func NewConcurrentCache(maxBytes uint64) ConcurrentCache {
	return ConcurrentCache{
		cacheMaxBytes: maxBytes,
		cl:            newConcurrentList(),
		cm:            newConcurrentMap(),
	}
}

func (c *ConcurrentCache) Add(key string, v view.ByteView) {
	n := &node{prev: nil, next: nil, entry: entry{key: key, data: v}}
	c.cm.set(key, n)
	c.cl.enqueue(n)
}

func (c *ConcurrentCache) Get(key string) (v view.ByteView, ok bool) {
	n, ok := c.cm.get(key)
	c.cl.delete(n)
	c.cl.enqueue(n)
	go c.RemoveOldest()
	return n.data, ok
}

func (c *ConcurrentCache) RemoveOldest() {
	for c.cacheMaxBytes != 0 && c.cacheMaxBytes < c.cl.usedMemorySize() {
		n := c.cl.dequeue()
		c.cm.delete(n.key)
	}
}

func (c *ConcurrentCache) KeyCount() uint64 {
	return c.cl.keyCount()
}

func (c *ConcurrentCache) UsedMemorySize() uint64 {
	return c.cl.usedMemorySize()
}
