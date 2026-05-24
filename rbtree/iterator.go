package rbtree

// Iterator is a forward/bidirectional cursor over Tree entries.
// The zero value is an invalid iterator and serves as the "end" sentinel.
// Iterators are stable across mutations that don't move the pointed-at
// node: Next-of-Erased-self is undefined, but other iterators survive an
// Erase. Inserts do not move existing nodes.
type Iterator[K, V any] struct {
	node *node[K, V]
}

func (it Iterator[K, V]) IsValid() bool { return it.node != nil }

// Key is undefined when !IsValid().
func (it Iterator[K, V]) Key() K { return it.node.key }

// Value is undefined when !IsValid().
func (it Iterator[K, V]) Value() V { return it.node.value }

// SetValue is undefined when !IsValid().
func (it Iterator[K, V]) SetValue(v V) { it.node.value = v }

// Next advances to the in-order successor; advancing past the last entry yields !IsValid().
func (it Iterator[K, V]) Next() Iterator[K, V] {
	if it.node != nil {
		it.node = successor(it.node)
	}
	return it
}

// Prev moves to the in-order predecessor; moving before the first entry yields !IsValid().
func (it Iterator[K, V]) Prev() Iterator[K, V] {
	if it.node != nil {
		it.node = predecessor(it.node)
	}
	return it
}

// Equal compares iterator identity (same underlying node), not key equality.
func (it Iterator[K, V]) Equal(other Iterator[K, V]) bool {
	return it.node == other.node
}
