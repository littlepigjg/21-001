package util

import "sort"

// StringSet 是字符串集合，基于 map 实现，保证元素唯一。
type StringSet struct {
	m map[string]struct{}
}

// NewStringSet 创建一个空字符串集合。
func NewStringSet() *StringSet {
	return &StringSet{m: make(map[string]struct{})}
}

// NewStringSetFrom 从切片创建一个字符串集合。
func NewStringSetFrom(list []string) *StringSet {
	s := NewStringSet()
	for _, v := range list {
		s.Add(v)
	}
	return s
}

// Add 向集合添加一个元素。
func (s *StringSet) Add(v string) {
	if s.m == nil {
		s.m = make(map[string]struct{})
	}
	s.m[v] = struct{}{}
}

// Remove 从集合移除一个元素。
func (s *StringSet) Remove(v string) {
	delete(s.m, v)
}

// Contains 判断集合是否包含指定元素。
func (s *StringSet) Contains(v string) bool {
	if s == nil || s.m == nil {
		return false
	}
	_, ok := s.m[v]
	return ok
}

// Size 返回集合元素数量。
func (s *StringSet) Size() int {
	if s == nil {
		return 0
	}
	return len(s.m)
}

// Slice 返回集合元素的排序切片。
func (s *StringSet) Slice() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.m))
	for v := range s.m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// Union 返回与另一个集合的并集。
func (s *StringSet) Union(other *StringSet) *StringSet {
	out := NewStringSet()
	for v := range s.m {
		out.Add(v)
	}
	if other != nil {
		for v := range other.m {
			out.Add(v)
		}
	}
	return out
}

// Intersect 返回与另一个集合的交集。
func (s *StringSet) Intersect(other *StringSet) *StringSet {
	out := NewStringSet()
	if other == nil {
		return out
	}
	for v := range s.m {
		if other.Contains(v) {
			out.Add(v)
		}
	}
	return out
}
