package util

import "sort"

type StringSet struct {
	m map[string]struct{}
}

func NewStringSet() *StringSet {
	return &StringSet{m: make(map[string]struct{})}
}

func NewStringSetFrom(list []string) *StringSet {
	s := NewStringSet()
	for _, v := range list {
		s.Add(v)
	}
	return s
}

func (s *StringSet) Add(v string) {
	if s.m == nil {
		s.m = make(map[string]struct{})
	}
	s.m[v] = struct{}{}
}

func (s *StringSet) Remove(v string) {
	delete(s.m, v)
}

func (s *StringSet) Contains(v string) bool {
	if s == nil || s.m == nil {
		return false
	}
	_, ok := s.m[v]
	return ok
}

func (s *StringSet) Size() int {
	if s == nil {
		return 0
	}
	return len(s.m)
}

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
