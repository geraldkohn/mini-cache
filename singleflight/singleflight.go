package singleflight

import (
	"sync"
)

// 存放要返回的数据和WaitGroup
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error

	// 异步调用的通道
	chans []chan<- Result
}

// 传递异步调用的结果
type Result struct {
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex // protect m
	m  map[string]*call
}

// Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
// 同步调用
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		// 正在被其他线程调用，等待结果
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	// 没有调用过
	c := new(call)
	// 线程开始, 加锁, 表示正在调用
	c.wg.Add(1)
	// 添加k-v
	g.m[key] = c
	g.mu.Unlock()

	// 调用fn
	c.val, c.err = fn()
	// 线程执行结束, 释放锁. 其他等待线程可以继续执行, 获取当前线程的结果. 这样可以防止重复调用
	c.wg.Done()

	// 立刻删除调用记录, 防止出现读取旧数据的情况.
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}

// 异步调用
func (g *Group) DoChan(key string, fn func() (interface{}, error)) <-chan Result {
	ch := make(chan Result, 1)
	g.mu.Lock()
	// 不存在字典
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 正在调用
	if c, ok := g.m[key]; ok {
		c.chans = append(c.chans, ch)
		g.mu.Unlock()
		return ch   // 调用结束, 返回值
	}
	// 没有被调用过
	c := new(call) // 新增call
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn() // 执行fn, 得到值

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	// 正在监听的每个通道都发送一个信息
	for _, ch := range c.chans {
		ch <- Result{c.val, c.err}
	}
	return ch
}
