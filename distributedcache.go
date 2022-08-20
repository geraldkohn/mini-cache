package cache

import (
	"errors"
	"log"
	"sync"
)

/*
	负责与外部交互，控制缓存存储和获取的主流程
*/

/*
	接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

// （3）如果缓存不存在，应该从数据源（数据库）获取缓存添加到缓存中。为了保持可扩展性，这里设计了一个回调函数，当缓存不存在时，调用这个函数，得到源数据。

// Gettr 接口为一个key载入数据
type Gettr interface {
	Get(key string) ([]byte, error)
}

// 定义回调函数，实现Gettr接口
type GettrFunc func(key string) ([]byte, error)

// 实现Gettr接口
func (f GettrFunc) Get(key string) ([]byte, error) {
	return f(key)
}

/*
	函数类型实现某一个接口，称之为接口型函数，方便调用者在调用时既能传入函数作为参数，也能传入实现该接口的结构体作为参数。
*/

// Group是一个缓存的命名空间，每个Group用一个唯一的名称name。
type Group struct {
	// 名称
	name string
	// 缓存未命中时获取源数据的回调函数
	gettr Gettr
	// 并发控制缓存
	coreCache cache
}

var (
	mu     sync.Mutex
	groups = make(map[string]*Group)
)

// 创建Group的一个实例
func NewGroup(name string, cacheMaxBytes int64, gettr Gettr) *Group {
	if gettr == nil {
		panic("nil Gettr")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		gettr:     gettr,
		coreCache: cache{mu: &sync.Mutex{}, cacheMaxBytes: cacheMaxBytes},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) (*Group, bool) {
	mu.Lock()
	defer mu.Unlock()
	if g, ok := groups[name]; ok {
		return g, true
	}
	return nil, false
}

// 从指定Group的缓存中读取key值。
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, errors.New("key is required")
	}

	// (1)命中本地缓存
	if v, ok := g.coreCache.get(key); ok {
		log.Println("Cache Hit!")
		return v, nil
	}

	// 调用回调函数，获取本地数据库中的值，并写入缓存
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	return g.getFromLocalDB(key)
}

func (g *Group) getFromLocalDB(key string) (ByteView, error) {
	byteSlice, err := g.gettr.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	v := ByteView{b: byteSlice}
	g.populateCache(key, v)
	return v, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.coreCache.add(key, value)
}
