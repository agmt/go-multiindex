// Package rbtree is a generic red-black tree with an STL-style iterator.
//
// External callers work with Tree (storage + ordered operations) and
// Iterator (cursor). The internal node type is package-private.
package rbtree

import (
	"fmt"
	"iter"
)

// Comparator is a three-way comparison function: negative if a<b,
// zero if a==b, positive if a>b. Same shape as cmp.Compare.
type Comparator[K any] func(a, b K) int

// Tree is a self-balancing red-black binary search tree. Duplicate keys
// are permitted; equal keys appear in insertion order during traversal.
type Tree[K, V any] struct {
	root   *node[K, V]
	size   int
	keyCmp Comparator[K]
}

// New creates an empty Tree ordered by cmp.
func New[K, V any](cmp Comparator[K]) *Tree[K, V] {
	return &Tree[K, V]{keyCmp: cmp}
}

// Cmp invokes the tree's comparator. Useful for callers that need to
// compare keys against the tree's ordering (e.g. for range termination).
func (t *Tree[K, V]) Cmp(lhs, rhs K) int { return t.keyCmp(lhs, rhs) }

func (t *Tree[K, V]) Size() int { return t.size }

func (t *Tree[K, V]) Empty() bool { return t.size == 0 }

func (t *Tree[K, V]) Clear() {
	t.root = nil
	t.size = 0
}

// ---- iterator factories ----

// Begin returns an iterator at the minimum-key entry, or invalid if empty.
func (t *Tree[K, V]) Begin() Iterator[K, V] {
	if t.root == nil {
		return Iterator[K, V]{}
	}
	return Iterator[K, V]{node: minimum(t.root)}
}

// RBegin returns an iterator at the maximum-key entry, or invalid if empty.
func (t *Tree[K, V]) RBegin() Iterator[K, V] {
	if t.root == nil {
		return Iterator[K, V]{}
	}
	return Iterator[K, V]{node: maximum(t.root)}
}

// Find returns an iterator at the first entry with key==key (duplicates
// are stored in insertion order), or invalid if absent.
func (t *Tree[K, V]) Find(key K) Iterator[K, V] {
	return Iterator[K, V]{node: t.findFirstNode(key)}
}

// LowerBound returns an iterator at the first entry with key >= key.
func (t *Tree[K, V]) LowerBound(key K) Iterator[K, V] {
	return Iterator[K, V]{node: t.findLowerBoundNode(t.root, key)}
}

// UpperBound returns an iterator at the first entry with key > key.
func (t *Tree[K, V]) UpperBound(key K) Iterator[K, V] {
	return Iterator[K, V]{node: t.findUpperBoundNode(t.root, key)}
}

// ---- mutations ----

// Insert always adds a new entry; duplicate keys are permitted and
// appear in insertion order during traversal.
func (t *Tree[K, V]) Insert(key K, value V) Iterator[K, V] {
	x := t.root
	var y *node[K, V]
	for x != nil {
		y = x
		if t.keyCmp(key, x.key) < 0 {
			x = x.left
		} else {
			x = x.right
		}
	}

	z := &node[K, V]{parent: y, color: red, key: key, value: value}
	t.size++

	if y == nil {
		z.color = black
		t.root = z
		return Iterator[K, V]{node: z}
	} else if t.keyCmp(z.key, y.key) < 0 {
		y.left = z
	} else {
		y.right = z
	}
	t.rbInsertFixup(z)
	return Iterator[K, V]{node: z}
}

// Erase invalidates the passed iterator; other iterators remain valid.
func (t *Tree[K, V]) Erase(it Iterator[K, V]) {
	t.deleteNode(it.node)
}

// ---- traversal ----

// Traversal walks the tree in key order; returning false from visit stops iteration.
func (t *Tree[K, V]) Traversal(visit func(k K, v V) bool) {
	for n := t.firstNode(); n != nil; n = successor(n) {
		if !visit(n.key, n.value) {
			return
		}
	}
}

// All returns a range-over-func sequence in key order.
func (t *Tree[K, V]) All() iter.Seq2[K, V] { return t.Traversal }

// verify checks that t satisfies all red-black invariants. Returns nil
// if valid, otherwise an error describing the first violated property.
// Package-private: only used by the rbtree test suite.
func (t *Tree[K, V]) verify() error {
	_, property, ok := t.check(t.root)
	if !ok {
		return fmt.Errorf("violated red-black property %d", property)
	}
	return nil
}

// ---- internals ----

func (t *Tree[K, V]) firstNode() *node[K, V] {
	if t.root == nil {
		return nil
	}
	return minimum(t.root)
}

func (t *Tree[K, V]) findFirstNode(key K) *node[K, V] {
	n := t.findLowerBoundNode(t.root, key)
	if n == nil || t.keyCmp(n.key, key) != 0 {
		return nil
	}
	return n
}

func (t *Tree[K, V]) findLowerBoundNode(x *node[K, V], key K) *node[K, V] {
	if x == nil {
		return nil
	}
	if t.keyCmp(key, x.key) <= 0 {
		ret := t.findLowerBoundNode(x.left, key)
		if ret == nil {
			return x
		}
		if t.keyCmp(ret.key, x.key) <= 0 {
			return ret
		}
		return x
	}
	return t.findLowerBoundNode(x.right, key)
}

func (t *Tree[K, V]) findUpperBoundNode(x *node[K, V], key K) *node[K, V] {
	if x == nil {
		return nil
	}
	if t.keyCmp(key, x.key) >= 0 {
		return t.findUpperBoundNode(x.right, key)
	}
	ret := t.findUpperBoundNode(x.left, key)
	if ret == nil {
		return x
	}
	if t.keyCmp(ret.key, x.key) <= 0 {
		return ret
	}
	return x
}

func (t *Tree[K, V]) deleteNode(z *node[K, V]) {
	if z == nil {
		return
	}

	var x, y *node[K, V]
	if z.left != nil && z.right != nil {
		y = successor(z)
	} else {
		y = z
	}

	if y.left != nil {
		x = y.left
	} else {
		x = y.right
	}

	xparent := y.parent
	if x != nil {
		x.parent = xparent
	}
	if y.parent == nil {
		t.root = x
	} else if y == y.parent.left {
		y.parent.left = x
	} else {
		y.parent.right = x
	}

	if y != z {
		z.key = y.key
		z.value = y.value
	}

	if y.color == black {
		t.rbDeleteFixup(x, xparent)
	}
	t.size--
}

func (t *Tree[K, V]) rbInsertFixup(z *node[K, V]) {
	var y *node[K, V]
	for z.parent != nil && z.parent.color == red {
		if z.parent == z.parent.parent.left {
			y = z.parent.parent.right
			if y != nil && y.color == red {
				z.parent.color = black
				y.color = black
				z.parent.parent.color = red
				z = z.parent.parent
			} else {
				if z == z.parent.right {
					z = z.parent
					t.leftRotate(z)
				}
				z.parent.color = black
				z.parent.parent.color = red
				t.rightRotate(z.parent.parent)
			}
		} else {
			y = z.parent.parent.left
			if y != nil && y.color == red {
				z.parent.color = black
				y.color = black
				z.parent.parent.color = red
				z = z.parent.parent
			} else {
				if z == z.parent.left {
					z = z.parent
					t.rightRotate(z)
				}
				z.parent.color = black
				z.parent.parent.color = red
				t.leftRotate(z.parent.parent)
			}
		}
	}
	t.root.color = black
}

func (t *Tree[K, V]) rbDeleteFixup(x, parent *node[K, V]) {
	for x != t.root && getColor(x) == black {
		if x != nil {
			parent = x.parent
		}
		if x == parent.left {
			x = t.rbFixupLeft(parent)
		} else {
			x = t.rbFixupRight(parent)
		}
	}
	if x != nil {
		x.color = black
	}
}

func (t *Tree[K, V]) rbFixupLeft(parent *node[K, V]) *node[K, V] {
	var x, w *node[K, V]
	w = parent.right
	if w.color == red {
		w.color = black
		parent.color = red
		t.leftRotate(parent)
		w = parent.right
	}
	if getColor(w.left) == black && getColor(w.right) == black {
		w.color = red
		x = parent
	} else {
		if getColor(w.right) == black {
			if w.left != nil {
				w.left.color = black
			}
			w.color = red
			t.rightRotate(w)
			w = parent.right
		}
		w.color = parent.color
		parent.color = black
		if w.right != nil {
			w.right.color = black
		}
		t.leftRotate(parent)
		x = t.root
	}
	return x
}

func (t *Tree[K, V]) rbFixupRight(parent *node[K, V]) *node[K, V] {
	var x, w *node[K, V]
	w = parent.left
	if w.color == red {
		w.color = black
		parent.color = red
		t.rightRotate(parent)
		w = parent.left
	}
	if getColor(w.left) == black && getColor(w.right) == black {
		w.color = red
		x = parent
	} else {
		if getColor(w.left) == black {
			if w.right != nil {
				w.right.color = black
			}
			w.color = red
			t.leftRotate(w)
			w = parent.left
		}
		w.color = parent.color
		parent.color = black
		if w.left != nil {
			w.left.color = black
		}
		t.rightRotate(parent)
		x = t.root
	}
	return x
}

func (t *Tree[K, V]) leftRotate(x *node[K, V]) {
	y := x.right
	x.right = y.left
	if y.left != nil {
		y.left.parent = x
	}
	y.parent = x.parent
	if x.parent == nil {
		t.root = y
	} else if x == x.parent.left {
		x.parent.left = y
	} else {
		x.parent.right = y
	}
	y.left = x
	x.parent = y
}

func (t *Tree[K, V]) rightRotate(x *node[K, V]) {
	y := x.left
	x.left = y.right
	if y.right != nil {
		y.right.parent = x
	}
	y.parent = x.parent
	if x.parent == nil {
		t.root = y
	} else if x == x.parent.right {
		x.parent.right = y
	} else {
		x.parent.left = y
	}
	y.right = x
	x.parent = y
}

// check validates red-black properties recursively. Returns
// (blackHeight, violatedProperty, ok).
//
// Properties:
//  1. Each node is either red or black.
//  2. The root is black.
//  3. All leaves (NIL) are black.
//  4. If a node is red, then both its children are black.
//  5. Every path from a node to a descendant NIL contains the same
//     number of black nodes.
func (t *Tree[K, V]) check(n *node[K, V]) (int, int, bool) {
	if n == nil {
		return 1, 0, true
	}
	if n == t.root && n.color == red {
		return 1, 2, false
	}
	leftBlack, property, ok := t.check(n.left)
	if !ok {
		return leftBlack, property, ok
	}
	rightBlack, property, ok := t.check(n.right)
	if !ok {
		return rightBlack, property, ok
	}
	if leftBlack != rightBlack {
		return leftBlack, 5, false
	}
	blackCount := leftBlack
	if n.color == red {
		if getColor(n.left) == red || getColor(n.right) == red {
			return 0, 4, false
		}
	} else {
		blackCount++
	}
	return blackCount, 0, true
}
