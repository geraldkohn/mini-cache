package concurrentcache

import (
	"container/list"
	"sync/atomic"
	"unsafe"
)

type concurrentList struct {
	length int
	head unsafe.Pointer
	tail unsafe.Pointer
}

type node struct {
	next unsafe.Pointer // 指向后一个节点
	data *list.Element  // 指向的list中的元素和
}

func newConcurrentList() *concurrentList {
	n := unsafe.Pointer(&node{})
	// 返回一个虚拟节点
	return &concurrentList{head: n, tail: n}
}

// 入队列
func (cl *concurrentList) enqueue(n *node) {
	for {
		tail := load(&cl.tail)
		tailNext := load(&tail.next)
		if tail == load(&cl.tail) { // 尾部还是尾部, 因为有可能有节点又来到尾部(并发)
			if tailNext == nil { // 尾部后面没有新数据入队列, 有可能有节点来到尾部(并发)
				if cas(&tail.next, tailNext, n) { // 将节点增加到队尾, 此时也有可能有节点来到尾部, 改变tailNext的值, 让它不是nil
					// 入队成功, 需要移动尾巴指针. 此时也存在并发问题, 但是如果有节点新来, 那么会再 if tailNext == nil 这步被判错(因为 cl.tail.next = n)
					// 然后走else, 将尾指针换成尾指针的next, 如果另一个线程的else先执行完毕, 下面的操作就会失败.
					// 但是无论成功还是失败, 尾指针已经再正确的位置上了. 因为要么另一个线程else成功, 更新尾指针的位置, 要么当前线程成功, 更新尾指针的位置.
					cas(&cl.tail, tail, n)
					return
				}
			} else { // 已经有数据来到尾部, 需要移动尾指针
				cas(&cl.tail, tail, tailNext)
			}
		}
	}
}

// 出队列
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
					return n
				}
			}
		}
	}
}

func load(p *unsafe.Pointer) *node {
	return (*node)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *node) (ok bool) {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}
