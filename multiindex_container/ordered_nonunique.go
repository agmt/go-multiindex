package multiindex_container

import (
	"iter"

	"github.com/agmt/go-multiindex"
	rbtree "github.com/agmt/go-multiindex/gostl_rbtree"
	"github.com/liyue201/gostl/utils/comparator"
)

type MultiIndexByOrderedNonUnique[K comparator.Ordered, V comparable] struct {
	Container *rbtree.RbTree[K, V]
	GetIndex  func(v V) K
}

func NewOrderedNonUnique[K comparator.Ordered, V comparable](
	getIndex func(v V) K,
) *MultiIndexByOrderedNonUnique[K, V] {
	mib := &MultiIndexByOrderedNonUnique[K, V]{
		Container: rbtree.New[K, V](comparator.OrderedTypeCmp),
		GetIndex:  getIndex,
	}
	return mib
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)
	return rbtree.NewIterator(t.Container.Insert(key, v))
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Find(key K) multiindex.ConstIterator[V] {
	return rbtree.NewIterator(t.Container.FindNode(key))
}

func (t *MultiIndexByOrderedNonUnique[K, V]) FindValue(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)

	for node := t.Container.FindLowerBoundNode(key); node != nil; node = node.Next() {
		if node.Value() == v {
			return rbtree.NewIterator(node)
		}
	}

	return nil
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Erase_Internal(it multiindex.ConstIterator[V]) {
	iter, ok := it.(*rbtree.RbTreeIterator[K, V])
	if !ok {
		panic("not iterator")
	}
	t.Container.DeleteIter(*iter)
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Size() int {
	return t.Container.Size()
}

func (t *MultiIndexByOrderedNonUnique[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		t.Container.Traversal(yield)
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) Where(k K) iter.Seq[V] {
	return func(yield func(V) bool) {
		for node := t.Container.FindLowerBoundNode(k); node != nil; node = node.Next() {
			if node.Key() != k {
				return
			}
			if !yield(node.Value()) {
				return
			}
		}
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) TraversalKV(visitor func(k K, v V) bool) {
	for k, v := range t.All() {
		if !visitor(k, v) {
			return
		}
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) TraversalValue(visitor func(v V) bool) {
	for _, v := range t.All() {
		if !visitor(v) {
			return
		}
	}
}

func (t *MultiIndexByOrderedNonUnique[K, V]) TraversalWithKey(k K, visitor func(v V) bool) {
	for v := range t.Where(k) {
		if !visitor(v) {
			return
		}
	}
}
