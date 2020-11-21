package consistenthash

import (
	"strconv"
	"testing"
)

func Test_Hashing(t *testing.T) {
	hash := New(3, func(data []byte) uint32 {
		i, _ := strconv.Atoi(string(data))
		return uint32(i)
	})

	// 一个6个虚拟节点组成哈希环
	// 2 4 6 12 14 16 22 24 26
	hash.Add("6", "4", "2")

	testCase := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCase {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// 添加一个真实节点 8
	// 则哈希环上增加值 8 18 28
	hash.Add("8")

	// 所以此时 27 应该映射到 真实节点 8
	testCase["27"] = "8"

	for k, v := range testCase {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
