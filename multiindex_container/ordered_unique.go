package multiindex_container

import (
	"github.com/agmt/go-multiindex"
	rbtree "github.com/agmt/go-multiindex/gostl_rbtree"
	"github.com/liyue201/gostl/utils/comparator"
)

type MultiIndexByOrderedUnique[K comparator.Ordered, V comparable] struct {
	MultiIndexByOrderedNonUnique[K, V]
}

func NewOrderedUnique[K comparator.Ordered, V comparable](
	main *multiindex.MultiIndex[V],
	getIndex func(v V) K,
) *MultiIndexByOrderedUnique[K, V] {
	mib := &MultiIndexByOrderedUnique[K, V]{
		MultiIndexByOrderedNonUnique[K, V]{
			Main:      main,
			Container: rbtree.New[K, V](comparator.OrderedTypeCmp),
			GetIndex:  getIndex,
		},
	}
	main.AddIndex(mib)
	return mib
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
