package multiindex

import "fmt"

type ConstIterator[V comparable] interface {
	IsValid() bool
	Value() V
}

type MultiIndexByI[V comparable] interface {
	Insert(V) ConstIterator[V]
	// FindFirst(key K) ConstIterator[V] // Unique for each type
	FindValue(v V) ConstIterator[V]
	Erase_Internal(ConstIterator[V]) // Erase only
	Size() int
	// TraversalKV(cb func(k K, v V))
	TraversalValue(cb func(v V) bool)
}

// All `V` should be different (or use *V)
type MultiIndex[V comparable] struct {
	MultiIndexBy []MultiIndexByI[V] // rbtree
}

func New[V comparable]() *MultiIndex[V] {
	return &MultiIndex[V]{
		MultiIndexBy: nil,
	}
}

func (m *MultiIndex[V]) Insert(v V) bool {
	if len(m.MultiIndexBy) == 0 {
		panic("multiindex has no indexes")
	}

	for i := 0; i < len(m.MultiIndexBy); i++ {
		cont := m.MultiIndexBy[i]
		it := cont.Insert(v)
		if it == nil || !it.IsValid() {
			// rollback
			for j := 0; j < i; j++ {
				it := m.MultiIndexBy[j].FindValue(v)
				m.MultiIndexBy[j].Erase_Internal(it)
			}
			return false
		}
	}

	return true
}

func (m *MultiIndex[V]) Erase(v V) {
	if len(m.MultiIndexBy) == 0 {
		panic("multiindex has no indexes")
	}

	for i := 0; i < len(m.MultiIndexBy); i++ {
		cont := m.MultiIndexBy[i]
		it := cont.FindValue(v)
		if it == nil || !it.IsValid() {
			// panic?
			continue
		}
		cont.Erase_Internal(it)
	}
}

func (m MultiIndex[V]) Size() int {
	if len(m.MultiIndexBy) == 0 {
		return 0
	}

	return m.MultiIndexBy[0].Size()
}

// ToDo: if `m` is non-empty, all existing elements should be indexed in `mib`
func (m *MultiIndex[V]) AddIndex(mib ...MultiIndexByI[V]) error {
	m.MultiIndexBy = append(m.MultiIndexBy, mib...)
	return nil
}

func (m MultiIndex[V]) Verify() (err error) {
	if len(m.MultiIndexBy) == 0 {
		return nil
	}
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	allValues := make(map[V]int)

	for i, cont := range m.MultiIndexBy {
		if i == 0 {
			cont.TraversalValue(func(v V) bool {
				_, ok := allValues[v]
				if ok {
					panic(fmt.Errorf("duplicate at %d: '%+v'", i, v))
				}
				allValues[v] = 1
				return true
			})
		} else {
			if len(allValues) != cont.Size() {
				panic(fmt.Errorf("wrong reported size at %d: %d != %d", i, len(allValues), cont.Size()))
			}
			cnt := 0
			cont.TraversalValue(func(v V) bool {
				_, ok := allValues[v]
				if !ok {
					panic(fmt.Errorf("exists only at %d: '%+v'", i, v))
				}
				cnt += 1
				return true
			})
			if len(allValues) != cont.Size() {
				panic(fmt.Errorf("wrong real size at %d: %d != %d", i, len(allValues), cont.Size()))
			}
		}
	}

	return nil
}
