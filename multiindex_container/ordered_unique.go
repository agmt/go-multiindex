package multiindex_container

import (
	"github.com/agmt/go-multiindex"
	rbtree "github.com/agmt/go-multiindex/gostl_rbtree"
	"github.com/liyue201/gostl/utils/comparator"
)

type MultiIndexByOrderedUnique[K comparable, V comparable] struct {
	MultiIndexByOrderedNonUnique[K, V]
}

func NewOrderedUnique[K comparator.Ordered, V comparable](
	getIndex func(v V) K,
) *MultiIndexByOrderedUnique[K, V] {
	mib := &MultiIndexByOrderedUnique[K, V]{
		MultiIndexByOrderedNonUnique[K, V]{
			Container: rbtree.New[K, V](comparator.OrderedTypeCmp),
			GetIndex:  getIndex,
		},
	}
	return mib
}

func NewOrderedUniqueWithCmp[K comparable, V comparable](
	getIndex func(v V) K,
	cmp func(a, b K) int,
) *MultiIndexByOrderedUnique[K, V] {
	mib := &MultiIndexByOrderedUnique[K, V]{
		MultiIndexByOrderedNonUnique[K, V]{
			Container: rbtree.New[K, V](cmp),
			GetIndex:  getIndex,
		},
	}
	return mib
}

func (t *MultiIndexByOrderedUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)
	if node := t.Container.FindNode(key); node != nil {
		return nil
	}
	return rbtree.NewIterator(t.Container.Insert(key, v))
}

func (t *MultiIndexByOrderedUnique[K, V]) InsertVWI(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)

	node := t.Container.FindNode(key)
	if node != nil {
		return nil
	}

	node = t.Container.Insert(key, v)
	return rbtree.NewIterator(node)
}
