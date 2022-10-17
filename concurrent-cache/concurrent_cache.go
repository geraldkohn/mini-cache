package concurrentcache

type concurrentCache struct {
	cacheMaxBytes uint64
	cl         *concurrentList
	cm          concurrentMap
}

type value interface {
	len() uint64
}

func NewConcurrentCache(maxBytes uint64) *concurrentCache {
	return &concurrentCache {
		cacheMaxBytes: maxBytes, 
		cl: newConcurrentList(), 
		cm: newConcurrentMap(),
	}
}

func (c *concurrentCache) Add(key string, v value) {
	n := &node{prev: nil, next: nil, data: v}
	c.cl.enqueue(n)
}

func (c *concurrentCache) Get(key string) (v value, ok bool) {
	n, ok := c.cm.get(key)
	c.cl.delete(n)
	c.cl.enqueue(n)
	go c.RemoveOldest()
	return n.data, ok
}

func (c *concurrentCache) RemoveOldest() {
	for c.cacheMaxBytes != 0 && c.cacheMaxBytes < c.cl.usedMemorySize() {
		c.cl.dequeue()
	}
}

func (c *concurrentCache) KeyCount() uint64 {
	return c.cl.keyCount()
}

func (c *concurrentCache) UsedMemorySize() uint64 {
	return c.cl.usedMemorySize()
}
