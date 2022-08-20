package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	// 允许使用的最大内存
	maxBytes int64
	// 当前已经使用的内存
	nBytes int64
	// 双向链表，存放节点数据
	ll *list.List
	// 字典（key与链表中节点地址的映射）
	cache map[string]*list.Element
	// optional and excuted when an entry is purged.
	// 可选的，当一个条目被清除时执行
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get looks up a key's value.
func (c *Cache) Get(key string) (value Value, ok bool) {
	// 在字典中找到节点地址
	if element, ok := c.cache[key]; ok {
		// 在字典中找到了节点地址
		// 将节点移动到链表首部
		c.ll.MoveToFront(element)
		// 节点值（k-v）
		kv := element.Value.(*entry)
		return kv.value, true
	}
	return
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value) {
	// 在字典中查找节点地址
	if ele, ok := c.cache[key]; ok {
		// 找到节点地址
		// 移动到链表首部
		c.ll.MoveToFront(ele)
		// 节点储存的（k-v对）
		kv := ele.Value.(*entry)
		// 更新使用的内存大小
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		// 更新节点储存的k-v对
		kv.value = value
	} else {
		// 在链表首部添加节点
		ele := c.ll.PushFront(&entry{key: key, value: value})
		// 更新字典
		c.cache[key] = ele
		// 更新使用的内存大小
		c.nBytes += int64(value.Len()) + int64(len(key))
	}
	// 内存达到设置的最大值，需要通过LRU策略移除节点
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		// 双向链表中移除节点
		c.ll.Remove(ele)
		// 字典中移除节点
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		// 更新使用的内存大小
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
