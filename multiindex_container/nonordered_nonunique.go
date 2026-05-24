package multiindex_container

import (
	"iter"

	"github.com/agmt/go-multiindex"
	"github.com/agmt/go-multiindex/swisstable"
)

// valueSet stores the set of values that share a key (non-unique index).
type valueSet[V comparable] = swisstable.Map[V, struct{}]

type MultiIndexByNonOrderedNonUnique[K comparable, V comparable] struct {
	Container *swisstable.Map[K, *valueSet[V]]
	GetIndex  func(v V) K
}

func NewNonOrderedNonUnique[K comparable, V comparable](
	getIndex func(v V) K,
) *MultiIndexByNonOrderedNonUnique[K, V] {
	return &MultiIndexByNonOrderedNonUnique[K, V]{
		Container: swisstable.New[K, *valueSet[V]](),
		GetIndex:  getIndex,
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) bucketFor(key K, createIfMissing bool) *valueSet[V] {
	if it := t.Container.Find(key); it.IsValid() {
		return it.Value()
	}
	if !createIfMissing {
		return nil
	}
	bucket := swisstable.New[V, struct{}]()
	t.Container.Insert(key, bucket)
	return bucket
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	bucket := t.bucketFor(t.GetIndex(v), true)
	bucket.Insert(v, struct{}{})
	return NewMapNonUniqueIterator(v)
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Find(key K) (it multiindex.ConstIterator[V]) {
	bucket := t.bucketFor(key, false)
	if bucket == nil {
		return
	}
	bi := bucket.Begin()
	if !bi.IsValid() {
		return
	}
	return NewMapNonUniqueIterator(bi.Key())
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) FindValue(vwi V) (it multiindex.ConstIterator[V]) {
	bucket := t.bucketFor(t.GetIndex(vwi), false)
	if bucket == nil || !bucket.Contains(vwi) {
		return
	}
	return NewMapNonUniqueIterator(vwi)
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Erase_Internal(cit multiindex.ConstIterator[V]) {
	it, ok := cit.(MapNonUniqueIterator[V])
	if !ok {
		panic("not iterator")
	}
	v := it.Value()
	key := t.GetIndex(v)
	bucket := t.bucketFor(key, false)
	if bucket == nil {
		return
	}
	bucket.Erase(v)
	if bucket.Empty() {
		t.Container.Erase(key)
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Size() int {
	sz := 0
	for _, bucket := range t.Container.All() {
		sz += bucket.Size()
	}
	return sz
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) TraversalKV(visitor func(k K, v V) bool) {
	for k, bucket := range t.Container.All() {
		keep := true
		for v := range bucket.All() {
			if !visitor(k, v) {
				keep = false
				break
			}
		}
		if !keep {
			return
		}
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) TraversalValue(visitor func(v V) bool) {
	for _, bucket := range t.Container.All() {
		keep := true
		for v := range bucket.All() {
			if !visitor(v) {
				keep = false
				break
			}
		}
		if !keep {
			return
		}
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) TraversalWithKey(k K, visitor func(v V) bool) {
	bucket := t.bucketFor(k, false)
	if bucket == nil {
		return
	}
	for v := range bucket.All() {
		if !visitor(v) {
			return
		}
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) All() iter.Seq2[K, V] {
	return t.TraversalKV
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Where(k K) iter.Seq[V] {
	return func(yield func(V) bool) {
		t.TraversalWithKey(k, yield)
	}
}

type MapNonUniqueIterator[V comparable] struct {
	ptr     V
	isValid bool
}

func NewMapNonUniqueIterator[V comparable](v V) MapNonUniqueIterator[V] {
	return MapNonUniqueIterator[V]{
		ptr:     v,
		isValid: true,
	}
}

func (it MapNonUniqueIterator[V]) IsValid() bool { return it.isValid }

func (it MapNonUniqueIterator[V]) Value() V { return it.ptr }
