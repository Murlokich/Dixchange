// Package avltree implements an ordered map as an AVL tree: the heights of the two subtrees of
// every node differ by at most one. Each node stores its height.
package avltree

import "cmp"

type node[K cmp.Ordered, V any] struct {
	key    K
	val    V
	left   *node[K, V]
	right  *node[K, V]
	parent *node[K, V]
	height int8 // 1 for a leaf
}

// AVLTree must be created with New; the zero value is not usable.
type AVLTree[K cmp.Ordered, V any] struct {
	root   *node[K, V]
	min    *node[K, V] // cached for O(1) Min, nil when empty
	max    *node[K, V] // cached for O(1) Max, nil when empty
	length int
}

func New[K cmp.Ordered, V any]() *AVLTree[K, V] {
	return &AVLTree[K, V]{}
}

func (t *AVLTree[K, V]) Len() int {
	return t.length
}

func (t *AVLTree[K, V]) Get(key K) (V, bool) {
	n := t.find(key)
	if n == nil {
		var zero V
		return zero, false
	}
	return n.val, true
}

// Put inserts the key, or replaces its value if it is already in the tree.
func (t *AVLTree[K, V]) Put(key K, val V) {
	var parent *node[K, V]
	x := t.root
	for x != nil {
		parent = x
		switch {
		case key < x.key:
			x = x.left
		case key > x.key:
			x = x.right
		default:
			x.val = val
			return
		}
	}

	n := &node[K, V]{key: key, val: val, parent: parent, height: 1}
	switch {
	case parent == nil:
		t.root = n
	case key < parent.key:
		parent.left = n
	default:
		parent.right = n
	}
	if t.min == nil || key < t.min.key {
		t.min = n
	}
	if t.max == nil || key > t.max.key {
		t.max = n
	}
	t.length++
	t.rebalanceUp(parent)
}

// Delete removes the key; false if it wasn't in the tree. A node with two children is replaced
// by its successor node itself (relinked, not copied), so no other node changes its key.
func (t *AVLTree[K, V]) Delete(key K) bool {
	z := t.find(key)
	if z == nil {
		return false
	}
	if z == t.min {
		t.min = successor(z)
	}
	if z == t.max {
		t.max = predecessor(z)
	}

	// the lowest node whose subtree lost a node; rebalancing starts there
	var start *node[K, V]
	switch {
	case z.left == nil:
		start = z.parent
		t.transplant(z, z.right)
	case z.right == nil:
		start = z.parent
		t.transplant(z, z.left)
	default:
		y := leftmost(z.right) // successor, takes z's position
		if y.parent == z {
			start = y
		} else {
			start = y.parent
			t.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
		}
		t.transplant(z, y)
		y.left = z.left
		y.left.parent = y
		y.height = z.height // the height of the position, so rebalanceUp sees what changed
	}
	t.length--
	t.rebalanceUp(start)
	return true
}

// Min returns the smallest key and its value; false if the tree is empty.
func (t *AVLTree[K, V]) Min() (K, V, bool) {
	return entry(t.min)
}

// Max returns the largest key and its value; false if the tree is empty.
func (t *AVLTree[K, V]) Max() (K, V, bool) {
	return entry(t.max)
}

func (t *AVLTree[K, V]) find(key K) *node[K, V] {
	x := t.root
	for x != nil {
		switch {
		case key < x.key:
			x = x.left
		case key > x.key:
			x = x.right
		default:
			return x
		}
	}
	return nil
}

// rebalanceUp walks from n to the root, updating heights and rotating where a node got out of
// balance. It stops early once a subtree keeps its height: nothing above it changes then.
func (t *AVLTree[K, V]) rebalanceUp(n *node[K, V]) {
	for n != nil {
		oldHeight := n.height
		n = t.rebalance(n)
		if n.height == oldHeight {
			return
		}
		n = n.parent
	}
}

// A node whose subtree heights differ by this much needs a rotation.
const outOfBalance = 2

// rebalance updates n's height and, if it is out of balance, rotates it back into balance.
// It returns the node now at n's position.
func (t *AVLTree[K, V]) rebalance(n *node[K, V]) *node[K, V] {
	updateHeight(n)
	switch balance(n) {
	case outOfBalance: // left-heavy
		if balance(n.left) < 0 { // left-right case: turn it into left-left first
			t.rotateLeft(n.left)
		}
		return t.rotateRight(n)
	case -outOfBalance: // right-heavy
		if balance(n.right) > 0 { // right-left case: turn it into right-right first
			t.rotateRight(n.right)
		}
		return t.rotateLeft(n)
	}
	return n
}

// rotateLeft makes x's right child y the root of the subtree, keeping the in-order sequence:
//
//	  x              y
//	 / \            / \
//	a   y    ->    x   c
//	   / \        / \
//	  b   c      a   b
func (t *AVLTree[K, V]) rotateLeft(x *node[K, V]) *node[K, V] {
	y := x.right
	x.right = y.left
	if y.left != nil {
		y.left.parent = x
	}
	t.transplant(x, y)
	y.left = x
	x.parent = y
	updateHeight(x)
	updateHeight(y)
	return y
}

// rotateRight makes y's left child x the root of the subtree, keeping the in-order sequence:
//
//	    y          x
//	   / \        / \
//	  x   c  ->  a   y
//	 / \            / \
//	a   b          b   c
func (t *AVLTree[K, V]) rotateRight(y *node[K, V]) *node[K, V] {
	x := y.left
	y.left = x.right
	if x.right != nil {
		x.right.parent = y
	}
	t.transplant(y, x)
	x.right = y
	y.parent = x
	updateHeight(y)
	updateHeight(x)
	return x
}

// transplant links v where u hangs: as the root or as the same child of u's parent.
// u's own children are left to the caller.
func (t *AVLTree[K, V]) transplant(u, v *node[K, V]) {
	switch {
	case u.parent == nil:
		t.root = v
	case u == u.parent.left:
		u.parent.left = v
	default:
		u.parent.right = v
	}
	if v != nil {
		v.parent = u.parent
	}
}

func height[K cmp.Ordered, V any](n *node[K, V]) int8 {
	if n == nil {
		return 0
	}
	return n.height
}

func updateHeight[K cmp.Ordered, V any](n *node[K, V]) {
	n.height = 1 + max(height(n.left), height(n.right))
}

// balance is the height of n's left subtree minus the height of its right subtree.
func balance[K cmp.Ordered, V any](n *node[K, V]) int8 {
	return height(n.left) - height(n.right)
}

func leftmost[K cmp.Ordered, V any](n *node[K, V]) *node[K, V] {
	for n.left != nil {
		n = n.left
	}
	return n
}

func rightmost[K cmp.Ordered, V any](n *node[K, V]) *node[K, V] {
	for n.right != nil {
		n = n.right
	}
	return n
}

func successor[K cmp.Ordered, V any](n *node[K, V]) *node[K, V] {
	if n.right != nil {
		return leftmost(n.right)
	}
	for n.parent != nil && n == n.parent.right {
		n = n.parent
	}
	return n.parent
}

func predecessor[K cmp.Ordered, V any](n *node[K, V]) *node[K, V] {
	if n.left != nil {
		return rightmost(n.left)
	}
	for n.parent != nil && n == n.parent.left {
		n = n.parent
	}
	return n.parent
}

func entry[K cmp.Ordered, V any](n *node[K, V]) (K, V, bool) {
	if n == nil {
		var key K
		var val V
		return key, val, false
	}
	return n.key, n.val, true
}
