// Package btree implements an ordered map as a B-tree (CLRS): keys and values are stored in all
// nodes, inserts and deletes restructure the tree top-down in a single pass.
package btree

import (
	"cmp"
	"slices"
)

type node[K cmp.Ordered, V any] struct {
	keys     []K           // sorted; capacity 2t-1, allocated once
	vals     []V           // vals[i] belongs to keys[i]
	children []*node[K, V] // nil for a leaf; otherwise len(keys)+1, capacity 2t
}

func (n *node[K, V]) leaf() bool {
	return n.children == nil
}

// search returns the index of the first key not less than key, scanning linearly, and whether
// it is the key itself.
func (n *node[K, V]) search(key K) (int, bool) {
	i := 0
	for i < len(n.keys) && n.keys[i] < key {
		i++
	}
	return i, i < len(n.keys) && n.keys[i] == key
}

// BTree must be created with New; the zero value is not usable.
type BTree[K cmp.Ordered, V any] struct {
	root   *node[K, V] // never nil; an empty tree is an empty leaf
	degree int
	length int
}

// The smallest valid minimum degree: with t = 1 a node could be left with no keys (t-1 = 0).
const minDegree = 2

// New creates an empty B-tree. degree is the minimum degree t: every node except the root holds
// between t-1 and 2t-1 keys, and an inner node has one child more than it has keys.
func New[K cmp.Ordered, V any](degree int) *BTree[K, V] {
	if degree < minDegree {
		panic("btree: degree must be at least 2")
	}
	t := &BTree[K, V]{degree: degree}
	t.root = t.newNode(true)
	return t
}

func (t *BTree[K, V]) Len() int {
	return t.length
}

func (t *BTree[K, V]) Get(key K) (V, bool) {
	n := t.root
	for {
		i, found := n.search(key)
		if found {
			return n.vals[i], true
		}
		if n.leaf() {
			var zero V
			return zero, false
		}
		n = n.children[i]
	}
}

// Put inserts the key, or replaces its value if it is already in the tree. A full node is split
// before the search descends into it, so the leaf that gets the key always has room.
func (t *BTree[K, V]) Put(key K, val V) {
	if len(t.root.keys) == t.maxKeys() {
		oldRoot := t.root
		t.root = t.newNode(false)
		t.root.children = append(t.root.children, oldRoot)
		t.splitChild(t.root, 0)
	}

	n := t.root
	for {
		i, found := n.search(key)
		if found {
			n.vals[i] = val
			return
		}
		if n.leaf() {
			n.keys = slices.Insert(n.keys, i, key)
			n.vals = slices.Insert(n.vals, i, val)
			t.length++
			return
		}
		if len(n.children[i].keys) == t.maxKeys() {
			t.splitChild(n, i)
			// the middle key of the split child moved up to n.keys[i]
			switch {
			case key == n.keys[i]:
				n.vals[i] = val
				return
			case key > n.keys[i]:
				i++
			}
		}
		n = n.children[i]
	}
}

// Delete removes the key; false if it wasn't in the tree. Before the search descends into a
// child, the child is given at least t keys (borrowed from a sibling or merged with one), so
// a key can always be removed from it without going below t-1.
func (t *BTree[K, V]) Delete(key K) bool {
	n := t.root
	for {
		i, found := n.search(key)
		if n.leaf() {
			t.shrinkRoot() // a merge on the way down can leave the root without keys
			if !found {
				return false
			}
			n.keys = slices.Delete(n.keys, i, i+1)
			n.vals = slices.Delete(n.vals, i, i+1)
			t.length--
			return true
		}

		if found {
			left, right := n.children[i], n.children[i+1]
			switch {
			case len(left.keys) >= t.degree:
				// replace the key with its predecessor, then delete the predecessor below
				p := rightmostLeaf(left)
				key = p.keys[len(p.keys)-1]
				n.keys[i], n.vals[i] = key, p.vals[len(p.vals)-1]
				n = left
			case len(right.keys) >= t.degree:
				// replace the key with its successor, then delete the successor below
				s := leftmostLeaf(right)
				key = s.keys[0]
				n.keys[i], n.vals[i] = key, s.vals[0]
				n = right
			default:
				// both children are minimal: merge them around the key, delete it from there
				n = t.merge(n, i)
			}
			continue
		}

		child := n.children[i]
		if len(child.keys) < t.degree {
			child = t.fill(n, i)
		}
		n = child
	}
}

// Min returns the smallest key and its value; false if the tree is empty.
func (t *BTree[K, V]) Min() (K, V, bool) {
	if t.length == 0 {
		var key K
		var val V
		return key, val, false
	}
	n := leftmostLeaf(t.root)
	return n.keys[0], n.vals[0], true
}

// Max returns the largest key and its value; false if the tree is empty.
func (t *BTree[K, V]) Max() (K, V, bool) {
	if t.length == 0 {
		var key K
		var val V
		return key, val, false
	}
	n := rightmostLeaf(t.root)
	return n.keys[len(n.keys)-1], n.vals[len(n.vals)-1], true
}

func (t *BTree[K, V]) maxKeys() int {
	return 2*t.degree - 1
}

func (t *BTree[K, V]) newNode(leaf bool) *node[K, V] {
	n := &node[K, V]{
		keys: make([]K, 0, t.maxKeys()),
		vals: make([]V, 0, t.maxKeys()),
	}
	if !leaf {
		n.children = make([]*node[K, V], 0, t.maxKeys()+1)
	}
	return n
}

// splitChild splits the full child y = parent.children[i] (2t-1 keys) into y and a new node z
// with t-1 keys each; the middle key moves up into parent at index i:
//
//	parent:   [ ... ]                [ ... m ... ]
//	              |         ->           /   \
//	y:     [ a  m  b ]               [ a ]   [ b ]    (a, b: t-1 keys each)
func (t *BTree[K, V]) splitChild(parent *node[K, V], i int) {
	y := parent.children[i]
	z := t.newNode(y.leaf())
	mid := t.degree - 1

	z.keys = append(z.keys, y.keys[mid+1:]...)
	z.vals = append(z.vals, y.vals[mid+1:]...)
	if !y.leaf() {
		z.children = append(z.children, y.children[mid+1:]...)
		clear(y.children[mid+1:])
		y.children = y.children[:mid+1]
	}

	parent.keys = slices.Insert(parent.keys, i, y.keys[mid])
	parent.vals = slices.Insert(parent.vals, i, y.vals[mid])
	parent.children = slices.Insert(parent.children, i+1, z)

	clear(y.keys[mid:])
	clear(y.vals[mid:])
	y.keys = y.keys[:mid]
	y.vals = y.vals[:mid]
}

// merge joins parent.children[i], the key parent.keys[i] and parent.children[i+1] into
// parent.children[i], and returns it. Both children have t-1 keys, so the result has 2t-1.
func (t *BTree[K, V]) merge(parent *node[K, V], i int) *node[K, V] {
	left, right := parent.children[i], parent.children[i+1]

	left.keys = append(left.keys, parent.keys[i])
	left.vals = append(left.vals, parent.vals[i])
	left.keys = append(left.keys, right.keys...)
	left.vals = append(left.vals, right.vals...)
	if !left.leaf() {
		left.children = append(left.children, right.children...)
	}

	parent.keys = slices.Delete(parent.keys, i, i+1)
	parent.vals = slices.Delete(parent.vals, i, i+1)
	rightIdx := i + 1
	parent.children = slices.Delete(parent.children, rightIdx, rightIdx+1)
	return left
}

// fill gives parent.children[i], which has t-1 keys, at least t keys before the search
// descends into it: it borrows a key through the parent from a sibling with t keys or more, or
// merges with a sibling. It returns the node to descend into.
func (t *BTree[K, V]) fill(parent *node[K, V], i int) *node[K, V] {
	child := parent.children[i]
	switch {
	case i > 0 && len(parent.children[i-1].keys) >= t.degree:
		// the parent's key moves down to the front of child, the left sibling's last key moves up
		left := parent.children[i-1]
		last := len(left.keys) - 1
		child.keys = slices.Insert(child.keys, 0, parent.keys[i-1])
		child.vals = slices.Insert(child.vals, 0, parent.vals[i-1])
		parent.keys[i-1], parent.vals[i-1] = left.keys[last], left.vals[last]
		left.keys = slices.Delete(left.keys, last, last+1)
		left.vals = slices.Delete(left.vals, last, last+1)
		if !child.leaf() {
			lastChild := last + 1
			child.children = slices.Insert(child.children, 0, left.children[lastChild])
			left.children = slices.Delete(left.children, lastChild, lastChild+1)
		}
		return child
	case i < len(parent.keys) && len(parent.children[i+1].keys) >= t.degree:
		// the parent's key moves down to the end of child, the right sibling's first key moves up
		right := parent.children[i+1]
		child.keys = append(child.keys, parent.keys[i])
		child.vals = append(child.vals, parent.vals[i])
		parent.keys[i], parent.vals[i] = right.keys[0], right.vals[0]
		right.keys = slices.Delete(right.keys, 0, 1)
		right.vals = slices.Delete(right.vals, 0, 1)
		if !child.leaf() {
			child.children = append(child.children, right.children[0])
			right.children = slices.Delete(right.children, 0, 1)
		}
		return child
	case i < len(parent.keys):
		return t.merge(parent, i)
	default:
		return t.merge(parent, i-1)
	}
}

// shrinkRoot drops a root left without keys by a merge of its last two children: the merged
// child becomes the root.
func (t *BTree[K, V]) shrinkRoot() {
	if len(t.root.keys) == 0 && !t.root.leaf() {
		t.root = t.root.children[0]
	}
}

func leftmostLeaf[K cmp.Ordered, V any](n *node[K, V]) *node[K, V] {
	for !n.leaf() {
		n = n.children[0]
	}
	return n
}

func rightmostLeaf[K cmp.Ordered, V any](n *node[K, V]) *node[K, V] {
	for !n.leaf() {
		n = n.children[len(n.children)-1]
	}
	return n
}
