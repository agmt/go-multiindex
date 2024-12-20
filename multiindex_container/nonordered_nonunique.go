package multiindex_container

import (
	"iter"

	"github.com/agmt/go-multiindex"
)

type MultiIndexByNonOrderedNonUnique[K comparable, V comparable] struct {
	Container map[K]map[V]bool
	GetIndex  func(v V) K
}

func NewNonOrderedNonUnique[K comparable, V comparable](
	getIndex func(v V) K,
) *MultiIndexByNonOrderedNonUnique[K, V] {
	mib := &MultiIndexByNonOrderedNonUnique[K, V]{
		Container: make(map[K]map[V]bool),
		GetIndex:  getIndex,
	}
	return mib
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Insert(v V) multiindex.ConstIterator[V] {
	key := t.GetIndex(v)
	rangeCont := t.Container[key]
	if rangeCont == nil {
		rangeCont = make(map[V]bool)
		t.Container[key] = rangeCont
	}
	rangeCont[v] = true
	return NewMapNonUniqueIterator(v)
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Find(key K) (iter multiindex.ConstIterator[V]) {
	rangeCont := t.Container[key]
	if rangeCont == nil {
		return
	}

	for v := range rangeCont {
		return NewMapNonUniqueIterator(v)
	}
	return
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) FindValue(vwi V) (iter multiindex.ConstIterator[V]) {
	key := t.GetIndex(vwi)
	rangeCont := t.Container[key]
	if rangeCont == nil {
		return
	}

	_, ok := rangeCont[vwi]
	if !ok {
		return
	}

	return NewMapNonUniqueIterator(vwi)
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Erase_Internal(it multiindex.ConstIterator[V]) {
	iter, ok := it.(MapNonUniqueIterator[V])
	if !ok {
		panic("wrong iterator")
	}
	key := t.GetIndex(iter.Value())

	subCont := t.Container[key]
	if subCont == nil {
		return
	}

	delete(subCont, iter.ptr)
	if len(subCont) == 0 {
		delete(t.Container, key)
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Size() int {
	sz := 0
	for _, subCont := range t.Container {
		sz += len(subCont)
	}
	return sz
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, cont := range t.Container {
			for v := range cont {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) Where(k K) iter.Seq[V] {
	return func(yield func(V) bool) {
		cont := t.Container[k]
		if cont == nil {
			return
		}
		for v := range cont {
			if !yield(v) {
				return
			}
		}
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) TraversalKV(visitor func(k K, v V) bool) {
	for k, v := range t.All() {
		if !visitor(k, v) {
			return
		}
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) TraversalValue(visitor func(vwi V) bool) {
	for _, v := range t.All() {
		if !visitor(v) {
			return
		}
	}
}

func (t *MultiIndexByNonOrderedNonUnique[K, V]) TraversalWithKey(k K, visitor func(v V) bool) {
	for v := range t.Where(k) {
		if !visitor(v) {
			return
		}
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

func (iter MapNonUniqueIterator[V]) IsValid() bool {
	return iter.isValid
}

func (iter MapNonUniqueIterator[V]) Value() V {
	return iter.ptr
}
