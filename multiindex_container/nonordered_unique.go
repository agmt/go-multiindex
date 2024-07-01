package multiindex_container

import (
	"github.com/agmt/go-multiindex"
	"github.com/liyue201/gostl/utils/comparator"
)

type MultiIndexByNonOrderedUnique[K comparable, V comparable] struct {
	Container map[K]V
	GetIndex  func(v V) K
}

func NewNonOrderedUnique[K comparator.Ordered, V comparable](
	getIndex func(v V) K,
) *MultiIndexByNonOrderedUnique[K, V] {
	mib := &MultiIndexByNonOrderedUnique[K, V]{
		Container: make(map[K]V),
		GetIndex:  getIndex,
	}
	return mib
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)
	_, exists := t.Container[key]
	if exists {
		return nil
	}
	t.Container[key] = v
	return MapIterator[K, V]{
		Key: key,
		Map: t.Container,
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Find(key K) multiindex.ConstIterator[V] {
	return MapIterator[K, V]{
		Key: key,
		Map: t.Container,
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) FindValue(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)

	return MapIterator[K, V]{
		Key: key,
		Map: t.Container,
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Erase_Internal(it multiindex.ConstIterator[V]) {
	iter, ok := it.(MapIterator[K, V])
	if !ok {
		panic("wrong iterator")
	}
	delete(t.Container, iter.Key)
}

func (t *MultiIndexByNonOrderedUnique[K, V]) Size() int {
	return len(t.Container)
}

func (t *MultiIndexByNonOrderedUnique[K, V]) TraversalKV(visitor func(k K, v V) bool) {
	for k, v := range t.Container {
		ok := visitor(k, v)
		if !ok {
			break
		}
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) TraversalValue(visitor func(v V) bool) {
	for _, v := range t.Container {
		ok := visitor(v)
		if !ok {
			break
		}
	}
}

func (t *MultiIndexByNonOrderedUnique[K, V]) TraversalWithKey(k K, visitor func(v V) bool) {
	v, ok := t.Container[k]
	if !ok {
		return
	}
	visitor(v)
}

type MapIterator[K comparable, V any] struct {
	Key K
	Map map[K]V
}

func (it MapIterator[K, V]) IsValid() bool {
	_, exists := it.Map[it.Key]
	return exists
}

func (it MapIterator[K, V]) Value() V {
	return it.Map[it.Key]
}
