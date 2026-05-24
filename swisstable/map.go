// Package swisstable is a portable-Go Swiss-Table hash map.
//
// Design follows Abseil and Go's runtime: groups of 8 slots, a packed
// uint64 control word per group, H1/H2 hash split, triangular probing.
// All match operations are pure-Go SWAR (no SIMD, no asm, no unsafe).
// Hashing uses hash/maphash.Comparable so quality and per-type cost
// match Go's builtin map.
//
// Public surface: Map (this file), Iterator (iterator.go).
// The group / slot / control-word machinery lives in group.go.
package swisstable

import (
	"hash/maphash"
	"math/bits"
)

// Map is an open-addressed Swiss-Table hash map. Not safe for concurrent use.
type Map[K comparable, V any] struct {
	seed       maphash.Seed
	groups     []group[K, V]
	groupMask  uint64 // len(groups)-1; len(groups) is always a power of two when non-zero
	size       int
	deadSlots  int // tombstones
	growthLeft int // remaining inserts before rehash
}

// New returns an empty Map. Storage is allocated lazily on first Put.
func New[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{seed: maphash.MakeSeed()}
}

// NewWithCapacity returns a Map preallocated to hold at least n entries
// without rehashing.
func NewWithCapacity[K comparable, V any](n int) *Map[K, V] {
	m := New[K, V]()
	m.Reserve(n)
	return m
}

func (m *Map[K, V]) Size() int { return m.size }

func (m *Map[K, V]) Empty() bool { return m.size == 0 }

// Clear removes all entries; underlying storage is retained.
func (m *Map[K, V]) Clear() {
	if m.groups == nil {
		return
	}
	for i := range m.groups {
		m.groups[i].ctrl = emptyCtrl
		var zero [groupSize]slot[K, V]
		m.groups[i].slots = zero
	}
	m.size = 0
	m.deadSlots = 0
	m.growthLeft = len(m.groups) * (groupSize - 1)
}

// Find returns an iterator at k, or an invalid iterator if absent.
func (m *Map[K, V]) Find(k K) Iterator[K, V] {
	invalid := Iterator[K, V]{m: m, groupIdx: len(m.groups)}
	if m.size == 0 {
		return invalid
	}
	h := maphash.Comparable(m.seed, k)
	h1 := h >> 7
	h2 := uint8(h & 0x7f)

	g := h1 & m.groupMask
	step := uint64(1)
	for {
		grp := &m.groups[g]
		ctrl := grp.ctrl

		matches := matchH2(ctrl, h2)
		for matches != 0 {
			i := bits.TrailingZeros64(matches) >> 3
			if grp.slots[i].key == k {
				return Iterator[K, V]{m: m, groupIdx: int(g), slotIdx: i} // #nosec G115 -- g <= groupMask fits in int
			}
			matches &= matches - 1
		}
		if matchEmpty(ctrl) != 0 {
			return invalid
		}
		g = (g + step) & m.groupMask
		step++
	}
}

func (m *Map[K, V]) Contains(k K) bool { return m.Find(k).IsValid() }

// Insert adds (k, v) only if k is absent (STL std::unordered_map::insert
// semantics). Returns an iterator at the entry (existing or new) and true
// if a new entry was created. For replace-on-duplicate semantics, use
// InsertOrAssign.
func (m *Map[K, V]) Insert(k K, v V) (Iterator[K, V], bool) {
	gi, si, found := m.findOrPrepare(k)
	if !found {
		h := maphash.Comparable(m.seed, k)
		m.commitInsert(gi, si, uint8(h&0x7f), k, v)
	}
	return Iterator[K, V]{m: m, groupIdx: gi, slotIdx: si}, !found
}

// InsertOrAssign adds (k, v), overwriting any existing value (C++17
// insert_or_assign). Returns the prior value (zero if absent) and whether
// a previous mapping was replaced.
func (m *Map[K, V]) InsertOrAssign(k K, v V) (oldV V, replaced bool) {
	gi, si, found := m.findOrPrepare(k)
	grp := &m.groups[gi]
	if found {
		oldV = grp.slots[si].val
		grp.slots[si].val = v
		return oldV, true
	}
	h := maphash.Comparable(m.seed, k)
	m.commitInsert(gi, si, uint8(h&0x7f), k, v)
	return oldV, false
}

// findOrPrepare probes for k. If found, returns its slot with found=true.
// If not found, returns a slot suitable for insertion (after handling any
// needed rehash) with found=false; the slot has growthLeft accounting
// still intact, so the caller can commit via commitInsert.
func (m *Map[K, V]) findOrPrepare(k K) (gi int, si int, found bool) {
	if m.groups == nil {
		m.initGroups(1)
	}
	h := maphash.Comparable(m.seed, k)
	h1 := h >> 7
	h2 := uint8(h & 0x7f)

	for {
		g := h1 & m.groupMask
		step := uint64(1)
		var insertG uint64
		var insertI int
		haveInsert := false

		for {
			grp := &m.groups[g]
			ctrl := grp.ctrl

			matches := matchH2(ctrl, h2)
			for matches != 0 {
				i := bits.TrailingZeros64(matches) >> 3
				if grp.slots[i].key == k {
					return int(g), i, true // #nosec G115 -- g <= groupMask fits in int
				}
				matches &= matches - 1
			}

			if !haveInsert {
				if empOrDel := matchEmptyOrDeleted(ctrl); empOrDel != 0 {
					insertG = g
					insertI = bits.TrailingZeros64(empOrDel) >> 3
					haveInsert = true
				}
			}

			if matchEmpty(ctrl) != 0 {
				break
			}

			g = (g + step) & m.groupMask
			step++
		}

		if !haveInsert || m.growthLeft == 0 {
			m.rehash()
			continue
		}

		return int(insertG), insertI, false // #nosec G115 -- insertG <= groupMask fits in int
	}
}

// commitInsert writes (k, v) at (gi, si) with the given H2 tag, updates size
// and growthLeft/deadSlots. Caller must guarantee the slot is reserved and
// growthLeft > 0 (typically by going through findOrPrepare).
func (m *Map[K, V]) commitInsert(gi, si int, h2 uint8, k K, v V) {
	grp := &m.groups[gi]
	wasDeleted := getCtrl(grp.ctrl, si) == ctrlDeleted
	setCtrl(&grp.ctrl, si, h2)
	grp.slots[si].key = k
	grp.slots[si].val = v
	m.size++
	if wasDeleted {
		m.deadSlots--
	} else {
		m.growthLeft--
	}
}

// Erase removes k. Returns the prior value (zero if absent) and whether k was present.
func (m *Map[K, V]) Erase(k K) (V, bool) {
	var zero V
	if m.size == 0 {
		return zero, false
	}
	h := maphash.Comparable(m.seed, k)
	h1 := h >> 7
	h2 := uint8(h & 0x7f)

	g := h1 & m.groupMask
	step := uint64(1)
	for {
		grp := &m.groups[g]
		ctrl := grp.ctrl

		matches := matchH2(ctrl, h2)
		for matches != 0 {
			i := bits.TrailingZeros64(matches) >> 3
			if grp.slots[i].key == k {
				oldV := grp.slots[i].val
				setCtrl(&grp.ctrl, i, ctrlDeleted)
				var zk K
				var zv V
				grp.slots[i].key = zk
				grp.slots[i].val = zv
				m.size--
				m.deadSlots++
				return oldV, true
			}
			matches &= matches - 1
		}
		if matchEmpty(ctrl) != 0 {
			return zero, false
		}
		g = (g + step) & m.groupMask
		step++
	}
}

// Reserve ensures capacity for at least n entries without further rehash.
func (m *Map[K, V]) Reserve(n int) {
	if n <= 0 {
		return
	}
	// maxFull = numGroups * (groupSize-1); need numGroups >= ceil(n / 7).
	needed := (n + groupSize - 2) / (groupSize - 1)
	if needed == 0 {
		needed = 1
	}
	target := 1
	for target < needed {
		target <<= 1
	}

	if m.groups == nil {
		m.initGroups(target)
		return
	}
	if len(m.groups) >= target {
		return
	}
	m.resize(target)
}

// ---- internals ----

func (m *Map[K, V]) initGroups(numGroups int) {
	m.groups = make([]group[K, V], numGroups)
	for i := range m.groups {
		m.groups[i].ctrl = emptyCtrl
	}
	m.groupMask = uint64(numGroups - 1) // #nosec G115 -- numGroups >= 1 enforced by callers
	m.size = 0
	m.deadSlots = 0
	// Max load = 7/8 of capacity.
	m.growthLeft = numGroups * (groupSize - 1)
}

func (m *Map[K, V]) rehash() {
	cur := len(m.groups)
	var target int
	// If >half real entries, grow 2x; otherwise same-size rehash to clean tombstones.
	if m.size*2 >= cur*groupSize {
		target = cur * 2
	} else {
		target = cur
	}
	if target == 0 {
		target = 1
	}
	m.resize(target)
}

func (m *Map[K, V]) resize(target int) {
	old := m.groups
	m.initGroups(target)
	for gi := range old {
		grp := &old[gi]
		full := matchFull(grp.ctrl)
		for full != 0 {
			i := bits.TrailingZeros64(full) >> 3
			m.insertNoCheck(grp.slots[i].key, grp.slots[i].val)
			full &= full - 1
		}
	}
}

// insertNoCheck inserts (k, v) assuming k is not already present and
// growthLeft > 0. Used during rehash from old storage.
func (m *Map[K, V]) insertNoCheck(k K, v V) {
	h := maphash.Comparable(m.seed, k)
	h1 := h >> 7
	h2 := uint8(h & 0x7f)

	g := h1 & m.groupMask
	step := uint64(1)
	for {
		grp := &m.groups[g]
		if empOrDel := matchEmptyOrDeleted(grp.ctrl); empOrDel != 0 {
			i := bits.TrailingZeros64(empOrDel) >> 3
			setCtrl(&grp.ctrl, i, h2)
			grp.slots[i].key = k
			grp.slots[i].val = v
			m.size++
			m.growthLeft--
			return
		}
		g = (g + step) & m.groupMask
		step++
	}
}
