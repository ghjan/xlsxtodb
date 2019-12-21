package set

import "sync"

type StringSet struct {
	m map[string]bool
	sync.RWMutex
}

func New() *StringSet {
	return &StringSet{
		m: map[string]bool{},
	}
}

func NewFromSlice(slice []string) *StringSet {
	object := &StringSet{
		m: map[string]bool{},
	}
	for _, item := range slice {
		object.Add(item)
	}

	return object
}

func (s *StringSet) Add(item string) {
	s.Lock()
	defer s.Unlock()
	s.m[item] = true
}
func (s *StringSet) Remove(item string) {
	s.Lock()
	s.Unlock()
	delete(s.m, item)
}
func (s *StringSet) Has(item string) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.m[item]
	return ok
}
func (s *StringSet) Len() int {
	return len(s.List())
}
func (s *StringSet) Clear() {
	s.Lock()
	defer s.Unlock()
	s.m = map[string]bool{}

}
func (s *StringSet) IsEmpty() bool {
	if s.Len() == 0 {
		return true
	}
	return false
}
func (s *StringSet) List() []string {
	s.RLock()
	defer s.RUnlock()
	list := []string{}
	for item := range s.m {
		list = append(list, item)
	}
	return list
}

func (s *StringSet) Union(other *StringSet) *StringSet {
	sNew := New()
	for _, item := range s.List() {
		sNew.Add(item)
	}
	for _, item := range other.List() {
		sNew.Add(item)
	}
	return sNew
}

func (s *StringSet) Difference(other *StringSet) *StringSet {
	sNew := New()
	for _, item := range s.List() {
		if !other.Has(item) {
			sNew.Add(item)
		}
	}
	return sNew
}

func (s *StringSet) Intersect(other *StringSet) *StringSet {
	sNew := New()
	for _, item := range s.List() {
		if other.Has(item) {
			sNew.Add(item)
		}
	}
	return sNew
}
