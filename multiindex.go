package multiindex

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

func (m *MultiIndex[V]) Reset() {
	if len(m.MultiIndexBy) == 0 {
		return
	}
	cont := m.MultiIndexBy[0]
	cont.TraversalValue(func(v V) bool {
		m.Erase(v)
		return true
	})
}

// ToDo: if `m` is non-empty, all existing elements should be indexed in `mib`
func (m *MultiIndex[V]) AddIndex(mib ...MultiIndexByI[V]) error {
	// If any of the provided indexes implement IsRbTree(), call it to allow
	// the underlying container to perform internal checks. Propagate any error.
	type isRbTree interface {
		IsRbTree() (bool, error)
	}

	for _, idx := range mib {
		if t, ok := idx.(isRbTree); ok {
			okRb, err := t.IsRbTree()
			if err != nil {
				return err
			}
			_ = okRb // currently we don't act on the boolean result
		}
	}

	m.MultiIndexBy = append(m.MultiIndexBy, mib...)
	return nil
}
