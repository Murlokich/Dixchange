// Package rbtree implements an ordered map as a red-black tree (CLRS, 3rd edition), with nil
// leaves instead of a sentinel node.
package rbtree

import "cmp"

type node[K cmp.Ordered, V any] struct {
	key    K
	val    V
	left   *node[K, V]
	right  *node[K, V]
	parent *node[K, V]
	red    bool
}

// RBTree must be created with New; the zero value is not usable.
type RBTree[K cmp.Ordered, V any] struct {
	root   *node[K, V]
	min    *node[K, V] // cached for O(1) Min, nil when empty
	max    *node[K, V] // cached for O(1) Max, nil when empty
	length int
}

func New[K cmp.Ordered, V any]() *RBTree[K, V] {
	return &RBTree[K, V]{}
}

func (t *RBTree[K, V]) Len() int {
	return t.length
}

func (t *RBTree[K, V]) Get(key K) (V, bool) {
	n := t.find(key)
	if n == nil {
		var zero V
		return zero, false
	}
	return n.val, true
}

// Put inserts the key, or replaces its value if it is already in the tree.
func (t *RBTree[K, V]) Put(key K, val V) {
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

	n := &node[K, V]{key: key, val: val, parent: parent, red: true}
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
	t.insertFixup(n)
}

// Delete removes the key; false if it wasn't in the tree. A node with two children is replaced
// by its successor node itself (relinked, not copied), so no other node changes its key.
func (t *RBTree[K, V]) Delete(key K) bool {
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

	// x takes the place of the node removed from its position; it can be nil, so its parent
	// is tracked separately for the fixup
	var x, xParent *node[K, V]
	removedRed := z.red
	switch {
	case z.left == nil:
		x, xParent = z.right, z.parent
		t.transplant(z, z.right)
	case z.right == nil:
		x, xParent = z.left, z.parent
		t.transplant(z, z.left)
	default:
		y := leftmost(z.right) // successor, takes z's position and color
		removedRed = y.red
		x = y.right
		if y.parent == z {
			xParent = y
		} else {
			xParent = y.parent
			t.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
		}
		t.transplant(z, y)
		y.left = z.left
		y.left.parent = y
		y.red = z.red
	}
	if !removedRed {
		t.deleteFixup(x, xParent)
	}
	t.length--
	return true
}

// Min returns the smallest key and its value; false if the tree is empty.
func (t *RBTree[K, V]) Min() (K, V, bool) {
	return entry(t.min)
}

// Max returns the largest key and its value; false if the tree is empty.
func (t *RBTree[K, V]) Max() (K, V, bool) {
	return entry(t.max)
}

func (t *RBTree[K, V]) find(key K) *node[K, V] {
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

// insertFixup restores the invariants after inserting the red node z: the only possible
// violation is z and its parent both being red.
//
//nolint:nestif // CLRS case analysis, the mirrored cases are kept side by side
func (t *RBTree[K, V]) insertFixup(z *node[K, V]) {
	for isRed(z.parent) {
		p := z.parent
		g := p.parent // exists: a red node is never the root
		if p == g.left {
			uncle := g.right
			if isRed(uncle) {
				p.red, uncle.red, g.red = false, false, true
				z = g
				continue
			}
			if z == p.right {
				z = p
				t.rotateLeft(z)
				p = z.parent
			}
			p.red, g.red = false, true
			t.rotateRight(g)
		} else {
			uncle := g.left
			if isRed(uncle) {
				p.red, uncle.red, g.red = false, false, true
				z = g
				continue
			}
			if z == p.left {
				z = p
				t.rotateRight(z)
				p = z.parent
			}
			p.red, g.red = false, true
			t.rotateLeft(g)
		}
	}
	t.root.red = false
}

// deleteFixup restores the invariants after a black node was removed above x: the path
// through x has one black node too few.
//
//nolint:nestif,gocognit // CLRS case analysis, the mirrored cases are kept side by side
func (t *RBTree[K, V]) deleteFixup(x, parent *node[K, V]) {
	for x != t.root && !isRed(x) {
		if x == parent.left {
			w := parent.right // sibling, never nil: its side has a black node more than x's
			if isRed(w) {
				w.red, parent.red = false, true
				t.rotateLeft(parent)
				w = parent.right
			}
			if !isRed(w.left) && !isRed(w.right) {
				w.red = true
				x, parent = parent, parent.parent
				continue
			}
			if !isRed(w.right) {
				w.left.red, w.red = false, true
				t.rotateRight(w)
				w = parent.right
			}
			w.red, parent.red, w.right.red = parent.red, false, false
			t.rotateLeft(parent)
			x = t.root
		} else {
			w := parent.left
			if isRed(w) {
				w.red, parent.red = false, true
				t.rotateRight(parent)
				w = parent.left
			}
			if !isRed(w.left) && !isRed(w.right) {
				w.red = true
				x, parent = parent, parent.parent
				continue
			}
			if !isRed(w.left) {
				w.right.red, w.red = false, true
				t.rotateLeft(w)
				w = parent.left
			}
			w.red, parent.red, w.left.red = parent.red, false, false
			t.rotateRight(parent)
			x = t.root
		}
	}
	if x != nil {
		x.red = false
	}
}

// rotateLeft makes x's right child y the root of the subtree, keeping the in-order sequence:
//
//	  x              y
//	 / \            / \
//	a   y    ->    x   c
//	   / \        / \
//	  b   c      a   b
func (t *RBTree[K, V]) rotateLeft(x *node[K, V]) {
	y := x.right
	x.right = y.left
	if y.left != nil {
		y.left.parent = x
	}
	t.transplant(x, y)
	y.left = x
	x.parent = y
}

// rotateRight makes y's left child x the root of the subtree, keeping the in-order sequence:
//
//	    y          x
//	   / \        / \
//	  x   c  ->  a   y
//	 / \            / \
//	a   b          b   c
func (t *RBTree[K, V]) rotateRight(y *node[K, V]) {
	x := y.left
	y.left = x.right
	if x.right != nil {
		x.right.parent = y
	}
	t.transplant(y, x)
	x.right = y
	y.parent = x
}

// transplant links v where u hangs: as the root or as the same child of u's parent.
// u's own children are left to the caller.
func (t *RBTree[K, V]) transplant(u, v *node[K, V]) {
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

func isRed[K cmp.Ordered, V any](n *node[K, V]) bool {
	return n != nil && n.red
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
