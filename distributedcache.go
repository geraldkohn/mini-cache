package cache

import (
	"errors"
	"log"
	concurrentcache "mini-cache/concurrent-cache"
	"mini-cache/singleflight"
	"mini-cache/view"
	"sync"

	pb "mini-cache/proto"
)

// 负责与外部交互，控制缓存存储和获取的主流程

/*
	接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

/*
	图一中的流程2：
	使用一致性哈希选择节点        是                                    是
    |-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
                    |  否                                    ↓  否
                    |----------------------------> 回退到本地节点处理。
*/

// （3）如果缓存不存在，应该从数据源（数据库）获取缓存添加到缓存中。为了保持可扩展性，这里设计了一个回调函数，当缓存不存在时，调用这个函数，得到源数据。

// 这里为一个函数式接口，只有一个函数实现的接口就可以被定义为函数式接口。即实现者不需要用额外的结构体来实现这个接口。
// 而是写一个函数，然后类型转换。
// 函数式接口可以让程序看起来更简洁
// Gettr 接口为一个key载入数据，这里是用来回调的。
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
	coreCache concurrentcache.ConcurrentCache
	// 查找远程节点
	peerPicker PeerPicker
	// 保证每一个key只会被获取一次
	loader *singleflight.Group
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
		coreCache: concurrentcache.NewConcurrentCache(uint64(cacheMaxBytes)),
		loader:    &singleflight.Group{},
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
func (g *Group) Get(key string) (view.ByteView, error) {
	if key == "" {
		return view.ByteView{}, errors.New("key is required")
	}

	// (1)命中本地缓存
	if v, ok := g.coreCache.Get(key); ok {
		log.Println("Cache Hit!")
		return v, nil
	}

	// 获取k-v，(2)(3)
	return g.load(key)
}

func (g *Group) load(key string) (view.ByteView, error) {
	// 封装
	viewI, err := g.loader.Do(key, func() (interface{}, error) {
		// 注册了集群节点
		if g.peerPicker != nil {
			var err error
			// key匹配到了集群节点
			if peer, ok := g.peerPicker.PickPeer(key); ok {
				// 从匹配的节点中获取了信息
				if value, err := g.getFromCluster(peer, key); err == nil {
					return value, nil
				}
				// 从集群获取失败
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getFromLocalDB(key)
	})

	if err == nil {
		return viewI.(view.ByteView), nil
	}

	return view.ByteView{}, err
}

// (2) 集群中获取数据
func (g *Group) getFromCluster(peer PeerServer, key string) (view.ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return view.ByteView{}, err
	}
	return view.ByteView{B: res.GetValue()}, nil
}

// （3）数据源（数据库）获取缓存添加到缓存中。
func (g *Group) getFromLocalDB(key string) (view.ByteView, error) {
	// 调用回调函数，获取本地数据库中的k-v值。
	byteSlice, err := g.gettr.Get(key)
	if err != nil {
		return view.ByteView{}, err
	}

	v := view.ByteView{B: byteSlice}
	g.populateCache(key, v)
	return v, nil
}

// 添加k-v
func (g *Group) populateCache(key string, value view.ByteView) {
	g.coreCache.Add(key, value)
}

// HTTPServer 实现了 PeerPicker，传递进来。
func (g *Group) RegisterPeers(peerPicker PeerPicker) {
	if g.peerPicker != nil {
		log.Println("RegisterPeerPicker called more than once")
	}
	g.peerPicker = peerPicker
}
