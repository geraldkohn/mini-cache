package singleflight

import "sync"

type call struct {
	wg sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex	// protect m
	m map[string]*call
}

// Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
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
	// 线程开始，加锁
	c.wg.Add(1)
	// 添加k-v，表示正在调用
	g.m[key] = c
	g.mu.Unlock()

	// 调用fn
	c.val, c.err = fn()
	// 线程执行结束，释放锁。其他等待线程可以继续执行，获取当前线程的结果。这样可以防止重复调用
	c.wg.Done()

	// 立刻删除调用记录，防止出现读取旧数据的情况。
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}