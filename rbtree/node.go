package rbtree

type color bool

const (
	red   color = false
	black color = true
)

// node is an internal red-black tree node. External callers manipulate
// the tree through Tree methods and Iterator, never through *node.
type node[K, V any] struct {
	parent, left, right *node[K, V]
	color               color
	key                 K
	value               V
}

func successor[K, V any](x *node[K, V]) *node[K, V] {
	if x.right != nil {
		return minimum(x.right)
	}
	y := x.parent
	for y != nil && x == y.right {
		x = y
		y = x.parent
	}
	return y
}

func predecessor[K, V any](x *node[K, V]) *node[K, V] {
	if x.left != nil {
		return maximum(x.left)
	}
	if x.parent == nil {
		return nil
	}
	if x.parent.right == x {
		return x.parent
	}
	for x.parent != nil && x.parent.left == x {
		x = x.parent
	}
	return x.parent
}

func minimum[K, V any](n *node[K, V]) *node[K, V] {
	for n.left != nil {
		n = n.left
	}
	return n
}

func maximum[K, V any](n *node[K, V]) *node[K, V] {
	for n.right != nil {
		n = n.right
	}
	return n
}

func getColor[K, V any](n *node[K, V]) color {
	if n == nil {
		return black
	}
	return n.color
}
