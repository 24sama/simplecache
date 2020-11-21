package lru

import "container/list"

// 最近最少使用，相对于仅考虑时间因素的 FIFO 和仅考虑访问频率的 LFU，LRU 算法可以认为是相对平衡的一种淘汰算法。
// LRU 认为，如果数据最近被访问过，那么将来被访问的概率也会更高。
// LRU 算法的实现非常简单，维护一个队列，如果某条记录被访问了，则移动到队尾，那么队首则是最近最少访问的数据，淘汰该条记录即可。

type Cache struct {
	maxBytes  int64                         // 允许最大使用的内存
	nBytes    int64                         // 当前使用的内存
	ll        *list.List                    // 双向链表记录存储的值
	cache     map[string]*list.Element      // map用来记录链表元素指针，便于查找
	OnEvicted func(key string, value Value) // 当某条记录被移除时的回调函数，可以为nil
}

// 双向列表存储节点的数据类型
// 链表中保存key值的好处在于，淘汰节点时，可以通过key从map中删除对应的映射
type entry struct {
	key   string
	value Value
}

// 节点的值可以是实现了Value接口的任意类型
// 该接口只有一个方法Len()，用于返回值所占用的内存大小
type Value interface {
	Len() int
}

// 初始化cache
func New(maxByte int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxByte,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找
// 根据lru的规则，查找需要分为两步
// 1. 通过map查找到对应的链表节点
// 2. 将该节点移至链表尾部
// 双向链表首尾是相对的，约定：队首back，队尾front
func (c *Cache) Get(key string) (value Value, ok bool) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		entry := elem.Value.(*entry)
		return entry.value, true
	}
	return
}

// 删除
// 缓存淘汰，即删除最近最少访问的节点（队首节点）
func (c *Cache) RemoveOldest() {
	elem := c.ll.Back()
	if elem != nil {
		c.ll.Remove(elem)
		entry := elem.Value.(*entry)
		delete(c.cache, entry.key)
		c.nBytes -= int64(len(entry.key)) + int64(entry.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(entry.key, entry.value)
		}
	}
}

// 新增/修改
// 如果键存在，则更新对应节点的值，并将该节点移至队尾
// 不存在则为新增，首先队尾添加新节点&entry{key, value}，并在map中添加key和节点的映射关系
// 更新c.nBytes，如果超过设定的最大值c.maxBytes，则移除最少访问的节点
func (c *Cache) Add(key string, value Value) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		entry := elem.Value.(*entry)
		entry.value = value
		c.nBytes += int64(value.Len()) - int64(entry.value.Len())
	} else {
		elem := c.ll.PushFront(&entry{key, value})
		c.cache[key] = elem
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
