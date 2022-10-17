package concurrentcache

import (
	"sync/atomic"
	"unsafe"
)

// lock-free list
// FIFO, 支持随机删除

type concurrentList struct {
	length    uint64 // 元素个数
	usedBytes uint64 // 使用的内存数量
	head      unsafe.Pointer
	tail      unsafe.Pointer
}

// 存放node中的数据
type entry struct {
	key  string
	data value
}

type node struct {
	prev unsafe.Pointer // 指向前一个节点
	next unsafe.Pointer // 指向后一个节点
	entry
}

func newConcurrentList() *concurrentList {
	n := unsafe.Pointer(&node{})
	// 返回一个虚拟节点
	return &concurrentList{head: n, tail: n}
}

// delete the given node from the queue
func (cl *concurrentList) delete(n *node) {
	// 需要防止head或者tail发生并发安全问题, 需要查看论文的写法.
	// 应该是将enqueue和dequeue整合进入就可以.
	// 直接调用入队出队方法会出现并发问题! 一个线程执行完后没有办法拦截其他执行一部分操作的线程.
	for {
		prev := load(&n.prev)
		next := load(&n.next)
		if cas(&prev.next, n, next) { // 尝试修改
			return // 修改成功
		}
	}
}

// enqueue puts the given node at the tail of the queue
func (cl *concurrentList) enqueue(n *node) {
	for {
		tail := load(&cl.tail)
		tailNext := load(&tail.next)
		if tail == load(&cl.tail) { // 尾部还是尾部, 因为有可能有节点又来到尾部(并发)
			if tailNext == nil { // 尾部后面没有新数据入队列, 有可能有节点来到尾部(并发)
				if cas(&tail.next, nil, n) { // 这里替换tail节点的next指针
					// cas(&n.prev, nil, tail) // 处理node.prev
					cas(&cl.tail, tail, n) // 如果执行失败, 那么说明其他线程已经移动了尾指针的位置
					store(&n.prev, tail)
					atomic.AddUint64(&cl.usedBytes, n.data.len())
					atomic.AddUint64(&cl.length, 1)
					return
				}
			} else { // 已经有数据来到尾部, 需要移动尾指针(并发问题)
				cas(&cl.tail, tail, tailNext)
			}
		}
	}
}

// dequeue removes and returns the node at the head of the queue.
// It returns nil if the queue is empty.
func (cl *concurrentList) dequeue() *node {
	for {
		head := load(&cl.head)
		tail := load(&cl.tail)
		headNext := load(&head.next)
		if head == load(&cl.head) { // head没有变
			if head == tail { // head 和 tail 一样
				if headNext == nil { // 说明是空队列
					return nil
				}
				// 如果headNext有值, 说明只是头指针没有调整
				// 调整尾指针指向头指针的下一个元素
				cas(&cl.tail, tail, headNext)
			} else {
				// 读取队列的数据
				n := headNext
				// 头指针移动到下一个, 如果没有compare成功, 说明有线程已经修改了头指针的位置. 就是意味着这个队头元素已经被取出了. 需要重试.
				if cas(&cl.head, head, headNext) {
					atomic.AddUint64(&cl.usedBytes, ^uint64(n.data.len()))
					atomic.AddUint64(&cl.length, ^uint64(0)) // length-1
					return n
				}
			}
		}
	}
}

// 队列的元素个数
func (cl *concurrentList) keyCount() uint64 {
	return cl.length
}

// 使用的内存
func (cl *concurrentList) usedMemorySize() uint64 {
	return cl.usedBytes
}

func store(p *unsafe.Pointer, new *node) {
	atomic.StorePointer(p, unsafe.Pointer(new))
}

func load(p *unsafe.Pointer) *node {
	return (*node)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *node) (ok bool) {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}
