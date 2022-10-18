package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash func 将数据映射到uint32上
type Hash func(data []byte) uint32

// type uint_32 []uint32

// func (u uint_32) Len() int           { return len(u) }
// func (u uint_32) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
// func (u uint_32) Less(i, j int) bool { return u[i] < u[j] }

type Pool struct {
	// Hash func
	hash Hash
	// 复制的节点数量
	replicas int
	// 映射到hash环上的虚拟节点的hash值，已经排序
	virtualNodes []int
	// 虚拟节点的Hash值映射到真实节点, 真实节点不放在环上
	// virtual nodes map to real nodes
	vMapToR map[int]string
}

func New(replicas int, fn Hash) *Pool {
	m := &Pool{
		replicas:     replicas,
		hash:         fn,
		vMapToR:      make(map[int]string),
		virtualNodes: make([]int, 0),
	}
	if m.hash == nil {
		// 默认的Hash算法
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点
func (p *Pool) Add(peers ...string) {
	for _, peer := range peers {
		for i := 0; i < p.replicas; i++ {
			// 通过添加编号的方式来区分虚拟节点
			hash := int(p.hash([]byte(strconv.Itoa(i) + peer)))
			p.vMapToR[hash] = peer
			p.virtualNodes = append(p.virtualNodes, int(hash))
		}
	}
	sort.Ints(p.virtualNodes)
}

// Get 获取在hash环上距离key最近的节点。
func (p *Pool) Get(key string) string {
	if len(p.virtualNodes) == 0 {
		// hash环上没有节点
		return ""
	}
	// key的hash值
	hash := int(p.hash([]byte(key)))
	// 二分搜索最近的虚拟节点的下标
	idx := sort.Search(len(p.virtualNodes), func(i int) bool { return p.virtualNodes[i] >= hash })

	// 将虚拟节点转换成为真实节点, 这里idx取余数的原因是可能 sort.Search() 搜索不到下标。然后就返回[0,n)的n。所以需要取余数，构成hash环
	peer := p.vMapToR[p.virtualNodes[idx%len(p.virtualNodes)]]

	return peer
}
