package multiindex_container

import (
	"cmp"
	"iter"

	"github.com/agmt/go-multiindex"
	"github.com/agmt/go-multiindex/rbtree"
)

type MultiIndexByOrderedNonUnique[K comparable, V comparable] struct {
	Container *rbtree.Tree[K, V]
	GetIndex  func(v V) K
}

func NewOrderedNonUnique[K cmp.Ordered, V comparable](
	getIndex func(v V) K,
) *MultiIndexByOrderedNonUnique[K, V] {
	return &MultiIndexByOrderedNonUnique[K, V]{
		Container: rbtree.New[K, V](cmp.Compare[K]),
		GetIndex:  getIndex,
	}
}

func NewOrderedNonUniqueWithCmp[K comparable, V comparable](
	getIndex func(v V) K,
	cmp func(a, b K) int,
) *MultiIndexByOrderedNonUnique[K, V] {
	return &MultiIndexByOrderedNonUnique[K, V]{
		Container: rbtree.New[K, V](cmp),
		GetIndex:  getIndex,
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	return t.Container.Insert(t.GetIndex(v), v)
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Find(key K) multiindex.ConstIterator[V] {
	return t.Container.Find(key)
}

func (t *MultiIndexByOrderedNonUnique[K, V]) FindValue(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)
	for it := t.Container.LowerBound(key); it.IsValid() && t.Container.Cmp(it.Key(), key) == 0; it = it.Next() {
		if it.Value() == v {
			return it
		}
	}
	return nil
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Erase_Internal(cit multiindex.ConstIterator[V]) {
	switch it := cit.(type) {
	case rbtree.Iterator[K, V]:
		t.Container.Erase(it)
	default:
		panic("not iterator")
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Size() int {
	return t.Container.Size()
}

func (t *MultiIndexByOrderedNonUnique[K, V]) TraversalRange(begin, end K, visitor func(k K, v V) bool) {
	for it := t.Container.LowerBound(begin); it.IsValid() && t.Container.Cmp(it.Key(), end) < 0; it = it.Next() {
		if !visitor(it.Key(), it.Value()) {
			return
		}
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) TraversalKV(visitor func(k K, v V) bool) {
	t.Container.Traversal(visitor)
}

func (t *MultiIndexByOrderedNonUnique[K, V]) TraversalValue(visitor func(v V) bool) {
	t.Container.Traversal(func(k K, v V) bool {
		return visitor(v)
	})
}

func (t *MultiIndexByOrderedNonUnique[K, V]) TraversalWithKey(k K, visitor func(v V) bool) {
	for it := t.Container.LowerBound(k); it.IsValid() && it.Key() == k; it = it.Next() {
		if !visitor(it.Value()) {
			return
		}
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) All() iter.Seq2[K, V] {
	return t.Container.All()
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Where(k K) iter.Seq[V] {
	return func(yield func(V) bool) {
		t.TraversalWithKey(k, yield)
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Range(begin, end K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		t.TraversalRange(begin, end, yield)
	}
}
