package swissmap

import (
	"iter"
	"math/bits"
)

// Iterator is an STL-style forward cursor over a Map. The zero value is
// invalid and serves as the "end" sentinel.
//
// Iterators are stable across mutations that do not trigger a rehash.
// Insert may invalidate all iterators (if it rehashes); Erase only
// invalidates the iterator pointing at the erased entry.
type Iterator[V any, K comparable] struct {
	m        *Map[V, K]
	groupIdx int
	slotIdx  int
}

// Begin returns an iterator at the first entry, or invalid if the map is empty.
func (m *Map[V, K]) Begin() Iterator[V, K] {
	return Iterator[V, K]{m: m}.advanced()
}

func (it Iterator[V, K]) IsValid() bool {
	return it.m != nil && it.groupIdx < len(it.m.groups)
}

// Key is undefined when !IsValid(). Direct slot read; no dispatch.
func (it Iterator[V, K]) Key() K {
	return it.m.groups[it.groupIdx].slots[it.slotIdx].key
}

// Value is undefined when !IsValid(). Direct slot read; no dispatch.
func (it Iterator[V, K]) Value() V {
	return it.m.groups[it.groupIdx].slots[it.slotIdx].val
}

// ValuePtr returns a pointer to the stored value. Undefined when !IsValid().
// The pointer is invalidated by any Insert that rehashes.
func (it Iterator[V, K]) ValuePtr() *V {
	return &it.m.groups[it.groupIdx].slots[it.slotIdx].val
}

// Next advances past the current entry; advancing past the last entry yields !IsValid().
func (it Iterator[V, K]) Next() Iterator[V, K] {
	it.slotIdx++
	return it.advanced()
}

func (it Iterator[V, K]) advanced() Iterator[V, K] {
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

// All returns a range-over-func sequence yielding values in unspecified order.
func (m *Map[V, K]) All() iter.Seq[V] {
	return func(yield func(V) bool) {
		for gi := range m.groups {
			grp := &m.groups[gi]
			full := matchFull(grp.ctrl)
			for full != 0 {
				i := bits.TrailingZeros64(full) >> 3
				if !yield(grp.slots[i].val) {
					return
				}
				full &= full - 1
			}
		}
	}
}

// AllKV returns a range-over-func sequence yielding (key, value) pairs in
// unspecified order.
func (m *Map[V, K]) AllKV() iter.Seq2[K, V] {
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
