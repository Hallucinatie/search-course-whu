package main

import (
	"fmt"

	"github.com/marcadamsge/gofuzzy/trie"
)

// 将 []map[string]any 中的  某一项(参数，字符串) 作为索引 构建快查哈希
func buildCourseIndex(data []map[string]any, indexKey string) (map[string]map[string]any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("数据为空，无法构建索引")
	}

	index := make(map[string]map[string]any)

	for _, item := range data {
		if value, exists := item[indexKey]; exists {
			keyStr, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("索引键 %s 的值不是字符串类型", indexKey)
			}
			index[keyStr] = item
		} else {
			return nil, fmt.Errorf("数据中缺少索引键 %s", indexKey)
		}
	}

	return index, nil
}

// 构建基于某字段的 Trie 索引
func buildCourseTrieIndex(data []map[string]any, indexKey string) (*trie.Trie[map[string]any], error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("数据为空，无法构建 Trie 索引")
	}

	t := trie.New[map[string]any]() // 创建 Trie，值类型为 map[string]any

	for _, item := range data {
		val, ok := item[indexKey]
		if !ok {
			return nil, fmt.Errorf("数据中缺少索引键 %s", indexKey)
		}

		keyStr, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("索引键 %s 的值不是字符串类型", indexKey)
		}

		t.Insert(keyStr, &item, nil) // 插入 Trie，值为 map 的指针
	}

	return t, nil
}
