package utils

import (
	"encoding/json"
)

func TransformStruct(in, out interface{}) error {
	bs, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, out)
}

func InAnySlice[T comparable](target T, list []T) bool {
	for _, t := range list {
		if t == target {
			return true
		}
	}
	return false
}

// 切片A和切片B是否有交集
func CommonInAB[T comparable](A, B []T) bool {
	var dstMap = make(map[T]bool)
	for _, b := range B {
		dstMap[b] = true
	}
	for _, a := range A {
		// 有1个相同就返回true
		if dstMap[a] {
			return true
		}
	}

	return false
}

func InSlice(s string, ss []string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}

func GroupAny[T any](items []T, size int) [][]T {
	var groups [][]T
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		groups = append(groups, items[i:end])
	}
	return groups
}
