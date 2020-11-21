package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 一致性哈希算法

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           // 采取依赖注入的方式，允许替换为自定义的Hash函数
	replicas int            // 虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点与真实节点的映射表
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	// 默认为ChecksumIEEE算法
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点的方法
// 允许传入0个或多个真实节点的名称
// 对于每一个真实节点，对应创建m.replicas个虚拟节点，虚拟节点的名称是strconv.Itoa(i) + key，即通过添加编号的方式区分不同虚拟节点
// 使用m.hash()计算节点哈希值，使用append(m.keys, hash)添加到hash环上
// 在hashMap中增加虚拟节点和真实节点的映射关系
// 根据hash值将环上的元素排序
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// 获取节点的方法
// 首先计算key的哈希值
// 顺时针找到第一个匹配虚拟节点的下标idx，即环上元素m.keys[i]的值第一个大于等于该key哈希值的元素
// 因为m.keys是一个环状结构，所以如果idx==len(m.keys)，说明应该选择m.keys[0]，所以用取余数的方式idx % len(m.keys)可以处理这种情况
// 通过hashMap映射得到真实节点
func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}
