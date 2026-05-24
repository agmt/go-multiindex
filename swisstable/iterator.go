package swisstable

import (
	"iter"
	"math/bits"
)

// Iterator is an STL-style forward cursor over a Map. The zero value is
// invalid and serves as the "end" sentinel.
//
// Iterators are stable across mutations that do not trigger a rehash.
// Insert may invalidate all iterators (if it rehashes); Erase only
// invalidates the iterator pointing at the erased entry. This matches
// the std::unordered_map guarantee.
type Iterator[K comparable, V any] struct {
	m        *Map[K, V]
	groupIdx int
	slotIdx  int
}

// Begin returns an iterator at the first entry, or invalid if the map is empty.
func (m *Map[K, V]) Begin() Iterator[K, V] {
	return Iterator[K, V]{m: m}.advanced()
}

func (it Iterator[K, V]) IsValid() bool {
	return it.m != nil && it.groupIdx < len(it.m.groups)
}

// Key is undefined when !IsValid().
func (it Iterator[K, V]) Key() K {
	return it.m.groups[it.groupIdx].slots[it.slotIdx].key
}

// Value is undefined when !IsValid().
func (it Iterator[K, V]) Value() V {
	return it.m.groups[it.groupIdx].slots[it.slotIdx].val
}

// Next advances past the current entry; advancing past the last entry yields !IsValid().
func (it Iterator[K, V]) Next() Iterator[K, V] {
	it.slotIdx++
	return it.advanced()
}

// advanced returns it shifted forward to the next live slot.
func (it Iterator[K, V]) advanced() Iterator[K, V] {
	for it.m != nil && it.groupIdx < len(it.m.groups) {
		full := matchFull(it.m.groups[it.groupIdx].ctrl)
		if it.slotIdx > 0 {
			full &= ^uint64(0) << (it.slotIdx * 8)
		}
		if full != 0 {
			it.slotIdx = bits.TrailingZeros64(full) >> 3
			return it
		}
		it.groupIdx++
		it.slotIdx = 0
	}
	return it
}

// All returns a range-over-func sequence of all entries in unspecified order.
// This is the fastest iteration path: the compiler inlines the yield closure
// with the SWAR matchFull loop body.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for gi := range m.groups {
			grp := &m.groups[gi]
			full := matchFull(grp.ctrl)
			for full != 0 {
				i := bits.TrailingZeros64(full) >> 3
				if !yield(grp.slots[i].key, grp.slots[i].val) {
					return
				}
				full &= full - 1
			}
		}
	}
}
