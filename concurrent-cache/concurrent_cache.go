package concurrentcache

type concurrentCache struct {
	cacheMaxBytes int64
	cList         *concurrentList
	cMap 		*concurrentMap
}

func NewConcurrentCache() *concurrentCache {

}

func (cache *concurrentCache) Add(key string, v value) {

}

func (cache *concurrentCache) Get(key string) (v value, ok bool) {

}

func (cache *concurrentCache) RemoveOldest() {

}

func (cache *concurrentCache) Len() int {
	return cache.cList.length
}