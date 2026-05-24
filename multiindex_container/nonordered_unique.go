package multiindex_container

import (
	"iter"

	"github.com/agmt/go-multiindex"
	"github.com/agmt/go-multiindex/swissmap"
)

type MultiIndexByNonOrderedUnique[K comparable, V comparable] struct {
	Container *swissmap.Map[V, K]
	GetIndex  func(v V) K
}

func NewNonOrderedUnique[K comparable, V comparable](
	getIndex func(v V) K,
) *MultiIndexByNonOrderedUnique[K, V] {
	return &MultiIndexByNonOrderedUnique[K, V]{
		Container: swissmap.New[V, K](getIndex),
		GetIndex:  getIndex,
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	it, isNew := t.Container.Insert(v)
	if !isNew {
		return nil
	}
	return it
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Find(key K) multiindex.ConstIterator[V] {
	return t.Container.Find(key)
}

func (t *MultiIndexByNonOrderedUnique[K, V]) FindValue(v V) multiindex.ConstIterator[V] {
	return t.Container.Find(t.GetIndex(v))
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Erase_Internal(cit multiindex.ConstIterator[V]) {
	it, ok := cit.(swissmap.Iterator[V, K])
	if !ok {
		panic("not iterator")
	}
	if !it.IsValid() {
		return
	}
	t.Container.Erase(it.Key())
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Size() int {
	return t.Container.Size()
}

func (t *MultiIndexByNonOrderedUnique[K, V]) TraversalKV(visitor func(k K, v V) bool) {
	for k, v := range t.Container.AllKV() {
		if !visitor(k, v) {
			return
		}
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) TraversalValue(visitor func(v V) bool) {
	for v := range t.Container.All() {
		if !visitor(v) {
			return
		}
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) TraversalWithKey(k K, visitor func(v V) bool) {
	if it := t.Container.Find(k); it.IsValid() {
		visitor(it.Value())
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) All() iter.Seq2[K, V] {
	return t.Container.AllKV()
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Where(k K) iter.Seq[V] {
	return func(yield func(V) bool) {
		t.TraversalWithKey(k, yield)
	}
}
