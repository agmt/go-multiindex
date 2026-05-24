package swissmap

import (
	"hash/maphash"
	"math/bits"
)

// Map is a Swiss-Table hash map keyed by a value extracted from each entry
// via the user-supplied keyOf closure. Not safe for concurrent use.
type Map[V any, K comparable] struct {
	seed       maphash.Seed
	keyOf      func(V) K
	groups     []group[K, V]
	groupMask  uint64 // len(groups)-1 when non-zero; mask for the H1 index
	size       int
	deadSlots  int // tombstones
	growthLeft int // remaining inserts before rehash
}

// New returns an empty Map that extracts each entry's index key via keyOf.
// keyOf takes V by value so that the entry doesn't escape to the heap on
// Insert. Storage is allocated lazily on first Insert.
func New[V any, K comparable](keyOf func(V) K) *Map[V, K] {
	return &Map[V, K]{
		seed:  maphash.MakeSeed(),
		keyOf: keyOf,
	}
}

// NewWithCapacity returns a Map preallocated for at least n entries
// without rehashing.
func NewWithCapacity[V any, K comparable](keyOf func(V) K, n int) *Map[V, K] {
	m := New[V, K](keyOf)
	m.Reserve(n)
	return m
}

func (m *Map[V, K]) Size() int { return m.size }

func (m *Map[V, K]) Empty() bool { return m.size == 0 }

// Clear removes all entries; underlying storage is retained.
func (m *Map[V, K]) Clear() {
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

// Find returns an iterator at the entry with the given key, or an invalid
// iterator if absent. Native K==K compare; no closure dispatch on the hot path.
func (m *Map[V, K]) Find(key K) Iterator[V, K] {
	invalid := Iterator[V, K]{m: m, groupIdx: len(m.groups)}
	if m.size == 0 {
		return invalid
	}
	h := maphash.Comparable(m.seed, key)
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
			if grp.slots[i].key == key {
				return Iterator[V, K]{m: m, groupIdx: int(g), slotIdx: i} // #nosec G115 -- g <= groupMask fits in int
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

func (m *Map[V, K]) Contains(key K) bool { return m.Find(key).IsValid() }

// Insert adds v only if no entry with the same key exists. Returns an
// iterator at the entry (existing or new) and true if a new entry was
// created. For replace-on-duplicate semantics use InsertOrAssign.
func (m *Map[V, K]) Insert(v V) (Iterator[V, K], bool) {
	key := m.keyOf(v)
	gi, si, found := m.findOrPrepare(key)
	if !found {
		h := maphash.Comparable(m.seed, key)
		m.commitInsert(gi, si, uint8(h&0x7f), key, v)
	}
	return Iterator[V, K]{m: m, groupIdx: gi, slotIdx: si}, !found
}

// InsertOrAssign adds v, replacing any existing entry with the same key.
// Returns the prior value (zero if absent) and whether a previous entry
// was replaced.
func (m *Map[V, K]) InsertOrAssign(v V) (oldV V, replaced bool) {
	key := m.keyOf(v)
	gi, si, found := m.findOrPrepare(key)
	grp := &m.groups[gi]
	if found {
		oldV = grp.slots[si].val
		grp.slots[si].val = v
		return oldV, true
	}
	h := maphash.Comparable(m.seed, key)
	m.commitInsert(gi, si, uint8(h&0x7f), key, v)
	return oldV, false
}

// Erase removes the entry with the given key. Returns the prior value
// (zero if absent) and whether an entry was present.
func (m *Map[V, K]) Erase(key K) (V, bool) {
	var zero V
	if m.size == 0 {
		return zero, false
	}
	h := maphash.Comparable(m.seed, key)
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
			if grp.slots[i].key == key {
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
func (m *Map[V, K]) Reserve(n int) {
	if n <= 0 {
		return
	}
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

func (m *Map[V, K]) initGroups(numGroups int) {
	m.groups = make([]group[K, V], numGroups)
	for i := range m.groups {
		m.groups[i].ctrl = emptyCtrl
	}
	m.groupMask = uint64(numGroups - 1) // #nosec G115 -- numGroups >= 1 enforced by callers
	m.size = 0
	m.deadSlots = 0
	m.growthLeft = numGroups * (groupSize - 1)
}

func (m *Map[V, K]) rehash() {
	cur := len(m.groups)
	var target int
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

func (m *Map[V, K]) resize(target int) {
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

func (m *Map[V, K]) insertNoCheck(key K, v V) {
	h := maphash.Comparable(m.seed, key)
	h1 := h >> 7
	h2 := uint8(h & 0x7f)

	g := h1 & m.groupMask
	step := uint64(1)
	for {
		grp := &m.groups[g]
		if empOrDel := matchEmptyOrDeleted(grp.ctrl); empOrDel != 0 {
			i := bits.TrailingZeros64(empOrDel) >> 3
			setCtrl(&grp.ctrl, i, h2)
			grp.slots[i].key = key
			grp.slots[i].val = v
			m.size++
			m.growthLeft--
			return
		}
		g = (g + step) & m.groupMask
		step++
	}
}

// findOrPrepare probes for an entry with the given key. If found, returns
// its slot. If not, returns a slot suitable for insertion (after handling
// any needed rehash); growthLeft is still intact so the caller may commit
// via commitInsert.
func (m *Map[V, K]) findOrPrepare(key K) (gi int, si int, found bool) {
	if m.groups == nil {
		m.initGroups(1)
	}
	h := maphash.Comparable(m.seed, key)
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
				if grp.slots[i].key == key {
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

func (m *Map[V, K]) commitInsert(gi, si int, h2 uint8, key K, v V) {
	grp := &m.groups[gi]
	wasDeleted := getCtrl(grp.ctrl, si) == ctrlDeleted
	setCtrl(&grp.ctrl, si, h2)
	grp.slots[si].key = key
	grp.slots[si].val = v
	m.size++
	if wasDeleted {
		m.deadSlots--
	} else {
		m.growthLeft--
	}
}
