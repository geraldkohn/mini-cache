package concurrentcache

import (
	"hash/crc32"
	"sync"
)

var shard_count = 32

type concurrentMap []*concurrentMapShard

// 分成shard_count个分片的map
type concurrentMapShard struct {
	items map[string]value
	sync.RWMutex
}

// 创建并发map
func newConcurrentMap() concurrentMap {
	m := make(concurrentMap, shard_count)
	for i := 0; i < shard_count; i++ {
		m[i] = &concurrentMapShard{items: make(map[string]value)}
	}
	return m
}

// 根据key计算分片索引
// For example, CRC32-Q, as defined by the following polynomial,
//
//	x³²+ x³¹+ x²⁴+ x²²+ x¹⁶+ x¹⁴+ x⁸+ x⁷+ x⁵+ x³+ x¹+ x⁰
//
// has the reversed notation 0b11010101100000101000001010000001, so the value
// that should be passed to MakeTable is 0xD5828281.
func (m concurrentMap) getShard(key string) *concurrentMapShard {
	table := crc32.MakeTable(0xD5828281)
	crc := crc32.Checksum([]byte(key), table)
	return m[crc%uint32(shard_count)]
}

func (m concurrentMap) set(key string, v value) {
	// 根据key计算分片
	shard := m.getShard(key)
	shard.Lock()
	// 对这个分片加锁, 执行业务操作
	shard.items[key] = v
	shard.Unlock()
}

func (m concurrentMap) get(key string) (v value, ok bool) {
	// 根据key计算分片
	shard := m.getShard(key)
	shard.RLock()
	v, ok = shard.items[key]
	shard.RUnlock()
	return
}
