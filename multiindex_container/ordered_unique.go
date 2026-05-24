package multiindex_container

import (
	"cmp"

	"github.com/agmt/go-multiindex"
	"github.com/agmt/go-multiindex/rbtree"
)

type MultiIndexByOrderedUnique[K comparable, V comparable] struct {
	MultiIndexByOrderedNonUnique[K, V]
}

func NewOrderedUnique[K cmp.Ordered, V comparable](
	getIndex func(v V) K,
) *MultiIndexByOrderedUnique[K, V] {
	return &MultiIndexByOrderedUnique[K, V]{
		MultiIndexByOrderedNonUnique[K, V]{
			Container: rbtree.New[K, V](cmp.Compare[K]),
			GetIndex:  getIndex,
		},
	}
}

func NewOrderedUniqueWithCmp[K comparable, V comparable](
	getIndex func(v V) K,
	cmp func(a, b K) int,
) *MultiIndexByOrderedUnique[K, V] {
	return &MultiIndexByOrderedUnique[K, V]{
		MultiIndexByOrderedNonUnique[K, V]{
			Container: rbtree.New[K, V](cmp),
			GetIndex:  getIndex,
		},
	}
}

func (t *MultiIndexByOrderedUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)
	if it := t.Container.Find(key); it.IsValid() {
		return nil
	}
	return t.Container.Insert(key, v)
}
