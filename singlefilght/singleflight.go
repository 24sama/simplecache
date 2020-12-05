package singlefilght

import "sync"

// 缓存雪崩：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。
// 缓存击穿：一个存在的key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时DB请求量大、压力骤增。
// 缓存穿透：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。
//
// 并发了 N 个请求 ?key=Tom，8003 节点向 8001 同时发起了 N 次请求。
// 假设对数据库的访问没有做任何限制的，很可能向数据库也发起 N 次请求，容易导致缓存击穿和穿透。
// 需要在高并发的情况下，多次外部请求在内部只会向数据库或远程端点发送一次请求

// call代表正在进行中或者y已经结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// singleflight主数据结构 管理不同key的请求
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do方法的作用是，针对相同的key，无论Do被调用多少次，函数fn都只会被调用一次
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         // 如果请求正在进行中， 则等待
		return c.val, c.err // 等待结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)  // 发起请求前加锁
	g.m[key] = c // group增加该key的映射，表示该key已经有请求在处理
	g.mu.Unlock()

	c.val, c.err = fn() // 调用fn处理请求
	c.wg.Done()         // 调用完成 解锁

	g.mu.Lock()
	delete(g.m, key) // 删除这波并发中记录的key映射
	g.mu.Unlock()

	return c.val, c.err
}
